package story

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/sebastiansucker/mAIrchen/backend/pkg/analysis"
	"github.com/sebastiansucker/mAIrchen/backend/pkg/config"
	"github.com/sebastiansucker/mAIrchen/backend/pkg/prompt"
)

// Story represents a generated story with metadata
type Story struct {
	Title           string   `json:"title"`
	Content         string   `json:"content"`
	Grundwortschatz []string `json:"grundwortschatz"`
	Model           string   `json:"model"`
	Provider        string   `json:"provider"`
	TokensUsed      int      `json:"tokens_used"`
	GenerationTime  float64  `json:"generation_time"`
}

// StreamCallbacks are invoked as the story is generated: OnTitle exactly
// once, then OnChunk zero or more times, before Generate returns.
type StreamCallbacks struct {
	OnTitle func(title string)
	OnChunk func(text string)
}

// Generator handles story generation
type Generator struct {
	config  *config.Config
	gwsDict map[string]string
}

// NewGenerator creates a new story generator
func NewGenerator(cfg *config.Config) *Generator {
	return &Generator{
		config:  cfg,
		gwsDict: analysis.ExtractGrundwortschatzWords(),
	}
}

// titleBufferLimit is how many characters we accumulate while looking for a
// "TITEL:" marker before giving up and treating everything seen so far as
// the story body (the model didn't follow the requested format).
const titleBufferLimit = 200

// Generate creates a story based on the given request, streaming the title
// and body text to the given callbacks as it arrives from the LLM.
func (g *Generator) Generate(ctx context.Context, req prompt.StoryRequest, cb StreamCallbacks) (*Story, error) {
	startTime := time.Now()

	fmt.Printf("\n=== Story Generation Start ===\n")
	fmt.Printf("Thema: %s, Länge: %d min, Klassenstufe: %s\n", req.Thema, req.Laenge, req.Klassenstufe)

	// Use configured model if not specified in request
	model := req.Model
	if model == "" {
		model = g.config.DefaultModel
	}
	fmt.Printf("Modell: %s\n", model)

	// Build prompts
	systemPrompt, userPrompt := prompt.BuildPrompt(req)

	// Create OpenAI client
	clientConfig := openai.DefaultConfig(g.config.OpenAIAPIKey)
	if g.config.OpenAIBaseURL != "" {
		clientConfig.BaseURL = g.config.OpenAIBaseURL
	}
	client := openai.NewClientWithConfig(clientConfig)

	stream, err := createChatCompletionStreamWithRetry(ctx, client, openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature:   0.8,
		MaxTokens:     8000,
		StreamOptions: &openai.StreamOptions{IncludeUsage: true},
	})
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer func() {
		_ = stream.Close()
	}()

	parser := newStreamParser(cb)
	tokensUsed := 0

	for {
		resp, recvErr := stream.Recv()
		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) {
				break
			}
			return nil, fmt.Errorf("stream interrupted: %w", recvErr)
		}

		if resp.Usage != nil {
			tokensUsed = resp.Usage.TotalTokens
		}

		if len(resp.Choices) == 0 {
			continue
		}

		parser.feed(resp.Choices[0].Delta.Content)
	}
	parser.finish()

	title := parser.title
	storyText := parser.fullStory.String()

	fmt.Printf("API Response - Tokens: %d, Zeichen: %d\n", tokensUsed, len(storyText))

	// Find Grundwortschatz words
	gwsWords := analysis.FindGrundwortschatzInText(storyText, g.gwsDict)

	generationTime := time.Since(startTime).Seconds()

	fmt.Printf("=== Generation abgeschlossen - Gesamt-Tokens: %d, Zeit: %.1fs ===\n\n", tokensUsed, generationTime)

	return &Story{
		Title:           title,
		Content:         storyText,
		Grundwortschatz: gwsWords,
		Model:           model,
		Provider:        g.config.AIProvider,
		TokensUsed:      tokensUsed,
		GenerationTime:  generationTime,
	}, nil
}

// streamRetryAttempts is how many extra tries are made to open the stream
// (i.e. up to this many retries after the first attempt) if the initial
// connection fails - e.g. a transient DNS lookup failure against a local
// Ollama host. Only the connection setup is retried, never anything after
// the first chunk has already reached the client, since re-sending from
// scratch at that point would show duplicated/inconsistent text.
const streamRetryAttempts = 2

var streamRetryDelay = 500 * time.Millisecond

func createChatCompletionStreamWithRetry(ctx context.Context, client *openai.Client, req openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error) {
	var lastErr error
	for attempt := 0; attempt <= streamRetryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(streamRetryDelay):
			}
			fmt.Printf("⚠️  Verbindungsaufbau fehlgeschlagen, Versuch %d/%d: %v\n", attempt+1, streamRetryAttempts+1, lastErr)
		}

		stream, err := client.CreateChatCompletionStream(ctx, req)
		if err == nil {
			return stream, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// streamParser consumes incremental Delta.Content fragments from the LLM
// stream and turns them into title/chunk callbacks plus a fully accumulated,
// cleaned story text (needed for Grundwortschatz analysis).
type streamParser struct {
	cb StreamCallbacks

	titleResolved bool
	title         string
	titleBuf      strings.Builder

	lineBuf   strings.Builder
	fullStory strings.Builder
	endeFound bool
}

func newStreamParser(cb StreamCallbacks) *streamParser {
	return &streamParser{cb: cb}
}

func (p *streamParser) feed(fragment string) {
	if fragment == "" {
		return
	}

	if !p.titleResolved {
		p.titleBuf.WriteString(fragment)
		rest, resolved := p.tryResolveTitle()
		if !resolved {
			return
		}
		fragment = rest
	}

	p.feedBody(fragment)
}

// tryResolveTitle checks the accumulated title buffer for a complete
// "TITEL: ...\n" line. Returns the body text that followed the title line
// (if any) and whether the title has now been resolved (found, or given up
// on after titleBufferLimit characters).
func (p *streamParser) tryResolveTitle() (string, bool) {
	buf := p.titleBuf.String()

	if title, rest, found := findTitelMarker(buf); found {
		p.title = title
		p.titleResolved = true
		p.cb.OnTitle(p.title)
		return rest, true
	}

	if len(buf) >= titleBufferLimit {
		p.title = "Ohne Titel"
		p.titleResolved = true
		p.cb.OnTitle(p.title)
		return buf, true
	}

	return "", false
}

func (p *streamParser) feedBody(fragment string) {
	for _, r := range fragment {
		if r == '\n' {
			p.flushLine(true)
			continue
		}
		p.lineBuf.WriteRune(r)
	}
}

// flushLine processes one completed (or, on finish(), final incomplete)
// line: strips markdown, checks for a standalone "ENDE" marker, and either
// emits it via OnChunk or (for ENDE) appends the decorative footer instead.
func (p *streamParser) flushLine(hadNewline bool) {
	if p.endeFound {
		p.lineBuf.Reset()
		return
	}

	line := removeMarkdownFormatting(p.lineBuf.String())
	p.lineBuf.Reset()

	if endeLineRegexp.MatchString(line) {
		p.endeFound = true
		p.fullStory.WriteString("\n\n" + strings.Repeat(" ", 25) + " ★ ENDE ★ " + strings.Repeat(" ", 25))
		return
	}

	out := line
	if hadNewline {
		out += "\n"
	}
	if out == "" {
		return
	}
	p.fullStory.WriteString(out)
	p.cb.OnChunk(out)
}

// finish must be called once the stream has ended (io.EOF). It resolves the
// title if none was ever found, and flushes any trailing partial line.
func (p *streamParser) finish() {
	if !p.titleResolved {
		p.title = "Ohne Titel"
		p.titleResolved = true
		p.cb.OnTitle(p.title)
		p.feedBody(p.titleBuf.String())
	}

	if p.lineBuf.Len() > 0 {
		p.flushLine(false)
	}
}

var endeLineRegexp = regexp.MustCompile(`(?i)^\s*ENDE\s*$`)

// findTitelMarker looks for a case-insensitive "TITEL:" marker followed by a
// newline in s. It returns the trimmed title text and everything after the
// title line, or found=false if no complete title line is present yet.
func findTitelMarker(s string) (title string, rest string, found bool) {
	upper := strings.ToUpper(s)
	idx := strings.Index(upper, "TITEL:")
	if idx < 0 {
		return "", "", false
	}

	afterMarker := s[idx+len("TITEL:"):]
	newlineIdx := strings.Index(afterMarker, "\n")
	if newlineIdx < 0 {
		return "", "", false
	}

	title = strings.TrimSpace(afterMarker[:newlineIdx])
	rest = strings.TrimLeft(afterMarker[newlineIdx+1:], " \t")
	return title, rest, true
}

// parseStory splits a complete (non-streamed) response into title and body.
// Kept for tests and as a reference implementation; the streaming path in
// streamParser reuses findTitelMarker for the same logic incrementally.
func parseStory(content string) (string, string) {
	if title, rest, found := findTitelMarker(content); found {
		return title, strings.TrimSpace(rest)
	}
	return "Ohne Titel", content
}

func removeMarkdownFormatting(text string) string {
	// Remove bold markers
	re := regexp.MustCompile(`\*\*(.*?)\*\*`)
	text = re.ReplaceAllString(text, "$1")

	// Remove italic markers
	re = regexp.MustCompile(`\*(.*?)\*`)
	text = re.ReplaceAllString(text, "$1")

	// Remove trailing markdown markers (e.g., "**Ende.**" -> "Ende.")
	text = strings.TrimRight(text, "*")

	// Remove common markdown patterns at the end
	re = regexp.MustCompile(`\*\*\s*$`)
	text = re.ReplaceAllString(text, "")

	return text
}

package story

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/sebastiansucker/mAIrchen/backend/pkg/config"
	"github.com/sebastiansucker/mAIrchen/backend/pkg/prompt"
)

// Test parseStory function
func TestParseStory(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectTitle string
		expectStory string
	}{
		{
			name: "Valid story format",
			input: `TITEL: Der kleine Hase
Es war einmal ein kleiner Hase.`,
			expectTitle: "Der kleine Hase",
			expectStory: "Es war einmal ein kleiner Hase.",
		},
		{
			name: "Story with markdown",
			input: `TITEL: **Der Fuchs**
Der Fuchs war *sehr* schlau.`,
			expectTitle: "**Der Fuchs**",
			expectStory: "Der Fuchs war *sehr* schlau.",
		},
		{
			name:        "Missing TITEL",
			input:       `Eine Geschichte ohne Titel.`,
			expectTitle: "Ohne Titel",
			expectStory: "Eine Geschichte ohne Titel.",
		},
		{
			name:        "Empty input",
			input:       "",
			expectTitle: "Ohne Titel",
			expectStory: "",
		},
		{
			name: "Story with multiple lines",
			input: `TITEL: Die Reise
Es war einmal.
Der Weg war lang.
Das Ende war schön.`,
			expectTitle: "Die Reise",
			expectStory: "Es war einmal.\nDer Weg war lang.\nDas Ende war schön.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			title, story := parseStory(tt.input)

			// Assert
			if title != tt.expectTitle {
				t.Errorf("Expected title '%s', got '%s'", tt.expectTitle, title)
			}
			if story != tt.expectStory {
				t.Errorf("Expected story '%s', got '%s'", tt.expectStory, story)
			}
		})
	}
}

func TestRemoveMarkdownFormatting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Bold text",
			input:    "**Hallo** Welt",
			expected: "Hallo Welt",
		},
		{
			name:     "Italic text",
			input:    "*Hallo* Welt",
			expected: "Hallo Welt",
		},
		{
			name:     "Mixed formatting",
			input:    "**Bold** und *italic* Text",
			expected: "Bold und italic Text",
		},
		{
			name:     "No formatting",
			input:    "Nur normaler Text",
			expected: "Nur normaler Text",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Multiple bold sections",
			input:    "**Eins** und **zwei** und **drei**",
			expected: "Eins und zwei und drei",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			result := removeMarkdownFormatting(tt.input)

			// Assert
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// feedFragments splits input into single-rune fragments to exercise the
// worst case for streamParser: a title/ENDE marker split across many chunks.
func feedFragments(p *streamParser, input string) {
	for _, r := range input {
		p.feed(string(r))
	}
	p.finish()
}

func TestStreamParser_TitleAndBody(t *testing.T) {
	var title string
	var chunks []string
	p := newStreamParser(StreamCallbacks{
		OnTitle: func(t string) { title = t },
		OnChunk: func(c string) { chunks = append(chunks, c) },
	})

	feedFragments(p, "TITEL: Der kleine Hase\nEs war einmal ein kleiner Hase.\nENDE")

	if title != "Der kleine Hase" {
		t.Errorf("expected title 'Der kleine Hase', got %q", title)
	}
	got := strings.Join(chunks, "")
	if !strings.Contains(got, "Es war einmal ein kleiner Hase.") {
		t.Errorf("expected body to contain the story text, got %q", got)
	}
	// The bare "ENDE" marker line itself must never reach the reader as a
	// standalone chunk - only the decorative "★ ENDE ★" footer should.
	for _, c := range chunks {
		if strings.TrimSpace(c) == "ENDE" {
			t.Errorf("raw ENDE marker must never be emitted as its own chunk, got chunks %q", chunks)
		}
	}
	if !strings.Contains(got, "★ ENDE ★") {
		t.Errorf("expected decorative ENDE footer to be emitted as a chunk, got %q", got)
	}
	if !strings.Contains(p.fullStory.String(), "★ ENDE ★") {
		t.Errorf("expected decorative ENDE footer in full story, got %q", p.fullStory.String())
	}
	// The footer must reach the reader identically to what gets scanned for
	// Grundwortschatz words, otherwise the backend can report a word (e.g.
	// "Ende" itself) that the frontend never actually displays.
	if got != p.fullStory.String() {
		t.Errorf("chunks and fullStory diverged: chunks %q vs fullStory %q", got, p.fullStory.String())
	}
}

func TestStreamParser_MissingTitel(t *testing.T) {
	var title string
	var chunks []string
	p := newStreamParser(StreamCallbacks{
		OnTitle: func(t string) { title = t },
		OnChunk: func(c string) { chunks = append(chunks, c) },
	})

	feedFragments(p, "Eine Geschichte ganz ohne Titelmarker.")

	if title != "Ohne Titel" {
		t.Errorf("expected fallback title 'Ohne Titel', got %q", title)
	}
	got := strings.Join(chunks, "")
	if !strings.Contains(got, "Eine Geschichte ganz ohne Titelmarker.") {
		t.Errorf("expected full text to be treated as body, got %q", got)
	}
}

func TestStreamParser_MarkdownStrippedPerLine(t *testing.T) {
	var chunks []string
	p := newStreamParser(StreamCallbacks{
		OnTitle: func(string) {},
		OnChunk: func(c string) { chunks = append(chunks, c) },
	})

	feedFragments(p, "TITEL: Titel\n**Fett** und *kursiv* Text.\nENDE")

	got := strings.Join(chunks, "")
	if strings.Contains(got, "*") {
		t.Errorf("expected markdown markers to be stripped, got %q", got)
	}
	if !strings.Contains(got, "Fett und kursiv Text.") {
		t.Errorf("expected cleaned text, got %q", got)
	}
}

func TestStreamParser_GivesUpOnTitleAfterBufferLimit(t *testing.T) {
	var title string
	var chunks []string
	p := newStreamParser(StreamCallbacks{
		OnTitle: func(s string) { title = s },
		OnChunk: func(c string) { chunks = append(chunks, c) },
	})

	// A model that ignores the format and writes prose straight away must not
	// have its output buffered indefinitely while waiting for a TITEL marker.
	longText := strings.Repeat("Ein langer Satz ohne Titelmarker. ", 10)
	if len(longText) <= titleBufferLimit {
		t.Fatalf("test input must exceed titleBufferLimit (%d), got %d", titleBufferLimit, len(longText))
	}
	p.feed(longText)

	// The title must already be resolved here - waiting for the stream to end
	// would leave the reader without a heading for the whole generation.
	if title != "Ohne Titel" {
		t.Errorf("expected the fallback title once the buffer limit is hit, got %q", title)
	}

	// Body text is only emitted on line boundaries, so it surfaces once the
	// line completes.
	p.feed("\n")
	if got := strings.Join(chunks, ""); !strings.Contains(got, "Ein langer Satz ohne Titelmarker.") {
		t.Errorf("expected the buffered text to be released as body, got %q", got)
	}

	p.finish()
	if !strings.Contains(p.fullStory.String(), "Ein langer Satz ohne Titelmarker.") {
		t.Errorf("expected the buffered text in the story, got %q", p.fullStory.String())
	}
}

func TestStreamParser_IgnoresEmptyFragments(t *testing.T) {
	var chunks []string
	p := newStreamParser(StreamCallbacks{
		OnTitle: func(string) {},
		OnChunk: func(c string) { chunks = append(chunks, c) },
	})

	p.feed("")
	p.feed("TITEL: Titel\n")
	p.feed("")
	p.feed("Text.\n")
	p.finish()

	if got := strings.Join(chunks, ""); got != "Text.\n" {
		t.Errorf("expected only the body line, got %q", got)
	}
}

func TestStreamParser_DropsEverythingAfterENDE(t *testing.T) {
	var chunks []string
	p := newStreamParser(StreamCallbacks{
		OnTitle: func(string) {},
		OnChunk: func(c string) { chunks = append(chunks, c) },
	})

	// Models sometimes keep talking after the ENDE marker (word lists,
	// commentary); none of it belongs in the story.
	feedFragments(p, "TITEL: Titel\nDie Geschichte.\nENDE\nVerwendete Wörter: Hund, Haus\n")

	got := strings.Join(chunks, "")
	if strings.Contains(got, "Verwendete Wörter") {
		t.Errorf("text after ENDE must be dropped, got %q", got)
	}
	if strings.Contains(p.fullStory.String(), "Verwendete Wörter") {
		t.Errorf("text after ENDE must not reach the story, got %q", p.fullStory.String())
	}
}

func TestStreamParser_SkipsBlankLines(t *testing.T) {
	var chunks []string
	p := newStreamParser(StreamCallbacks{
		OnTitle: func(string) {},
		OnChunk: func(c string) { chunks = append(chunks, c) },
	})

	// A trailing partial line that is empty after markdown stripping must not
	// produce an empty chunk.
	feedFragments(p, "TITEL: Titel\nText.\n**")

	for _, c := range chunks {
		if c == "" {
			t.Error("an empty chunk must never be emitted")
		}
	}
	if got := strings.Join(chunks, ""); got != "Text.\n" {
		t.Errorf("expected only the body line, got %q", got)
	}
}

func TestCreateChatCompletionStreamWithRetry_RetriesOnFailureThenSucceeds(t *testing.T) {
	origDelay := streamRetryDelay
	streamRetryDelay = time.Millisecond
	defer func() { streamRetryDelay = origDelay }()

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hi\"},\"index\":0,\"finish_reason\":\"\"}]}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	clientConfig := openai.DefaultConfig("test-key")
	clientConfig.BaseURL = server.URL
	client := openai.NewClientWithConfig(clientConfig)

	stream, err := createChatCompletionStreamWithRetry(context.Background(), client, openai.ChatCompletionRequest{
		Model:    "test-model",
		Messages: []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("expected retry to eventually succeed, got error: %v", err)
	}
	defer func() { _ = stream.Close() }()

	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Errorf("expected exactly 2 attempts (1 failure + 1 success), got %d", got)
	}
}

func TestCreateChatCompletionStreamWithRetry_GivesUpAfterMaxAttempts(t *testing.T) {
	origDelay := streamRetryDelay
	streamRetryDelay = time.Millisecond
	defer func() { streamRetryDelay = origDelay }()

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	clientConfig := openai.DefaultConfig("test-key")
	clientConfig.BaseURL = server.URL
	client := openai.NewClientWithConfig(clientConfig)

	_, err := createChatCompletionStreamWithRetry(context.Background(), client, openai.ChatCompletionRequest{
		Model:    "test-model",
		Messages: []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected an error after exhausting all retries")
	}

	if got := atomic.LoadInt32(&attempts); got != streamRetryAttempts+1 {
		t.Errorf("expected exactly %d attempts, got %d", streamRetryAttempts+1, got)
	}
}

// sseServer serves an OpenAI-compatible stream built from the given raw
// "data:" payloads, so a test can shape exactly what Generate() has to cope
// with (including malformed frames).
func sseServer(t *testing.T, payloads []string, capture *openai.ChatCompletionRequest) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if capture != nil {
			if err := json.NewDecoder(r.Body).Decode(capture); err != nil {
				t.Errorf("could not decode the outgoing request: %v", err)
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)
		for _, p := range payloads {
			_, _ = fmt.Fprintf(w, "data: %s\n\n", p)
			flusher.Flush()
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// contentFrames turns text into one SSE delta frame per line, mirroring how a
// provider dribbles a story back.
func contentFrames(text string) []string {
	var frames []string
	for _, line := range strings.SplitAfter(text, "\n") {
		if line == "" {
			continue
		}
		chunk, _ := json.Marshal(map[string]any{
			"choices": []map[string]any{{"index": 0, "delta": map[string]string{"content": line}}},
		})
		frames = append(frames, string(chunk))
	}
	return frames
}

func usageFrame(totalTokens int) string {
	frame, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{},
		"usage":   map[string]int{"total_tokens": totalTokens},
	})
	return string(frame)
}

func testConfig(baseURL string) *config.Config {
	return &config.Config{
		AIProvider:    "openai",
		OpenAIAPIKey:  "test-key",
		OpenAIBaseURL: baseURL,
		DefaultModel:  "default-model",
	}
}

func TestGenerate_ReturnsParsedStory(t *testing.T) {
	frames := append(contentFrames("TITEL: Der kleine Hase\nEs war einmal der kleine Hase im Wald.\nENDE\n"), usageFrame(1500), "[DONE]")
	server := sseServer(t, frames, nil)

	var titles, chunks []string
	generated, err := NewGenerator(testConfig(server.URL)).Generate(
		context.Background(),
		prompt.StoryRequest{Thema: "Mut", Laenge: 5, Klassenstufe: "12"},
		StreamCallbacks{
			OnTitle: func(s string) { titles = append(titles, s) },
			OnChunk: func(s string) { chunks = append(chunks, s) },
		},
	)
	if err != nil {
		t.Fatalf("expected the generation to succeed, got %v", err)
	}

	if generated.Title != "Der kleine Hase" {
		t.Errorf("expected the parsed title, got %q", generated.Title)
	}
	if len(titles) != 1 {
		t.Errorf("expected OnTitle to fire exactly once, got %d calls: %v", len(titles), titles)
	}
	if !strings.Contains(generated.Content, "Es war einmal der kleine Hase im Wald.") {
		t.Errorf("expected the body text in the content, got %q", generated.Content)
	}
	if strings.Contains(strings.Join(chunks, ""), "TITEL:") {
		t.Errorf("the title line must not be streamed as body, got %q", strings.Join(chunks, ""))
	}
	if generated.TokensUsed != 1500 {
		t.Errorf("expected 1500 tokens, got %d", generated.TokensUsed)
	}
	if generated.Provider != "openai" {
		t.Errorf("expected the provider to be carried over, got %q", generated.Provider)
	}
	if generated.GenerationTime < 0 {
		t.Errorf("expected a non-negative generation time, got %v", generated.GenerationTime)
	}

	// The Grundwortschatz scan runs over the assembled story, so it can only
	// work if the streaming path accumulated the text correctly.
	found := false
	for _, w := range generated.Grundwortschatz {
		if w == "Hase" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'Hase' among the Grundwortschatz matches, got %v", generated.Grundwortschatz)
	}
}

func TestGenerate_ModelSelection(t *testing.T) {
	tests := []struct {
		name        string
		requested   string
		expectModel string
	}{
		{name: "falls back to the configured model", requested: "", expectModel: "default-model"},
		{name: "request overrides the configured model", requested: "custom-model", expectModel: "custom-model"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sent openai.ChatCompletionRequest
			frames := append(contentFrames("TITEL: T\nText.\n"), usageFrame(10), "[DONE]")
			server := sseServer(t, frames, &sent)

			generated, err := NewGenerator(testConfig(server.URL)).Generate(
				context.Background(),
				prompt.StoryRequest{Thema: "Mut", Laenge: 5, Klassenstufe: "12", Model: tt.requested},
				StreamCallbacks{OnTitle: func(string) {}, OnChunk: func(string) {}},
			)
			if err != nil {
				t.Fatalf("expected the generation to succeed, got %v", err)
			}

			if sent.Model != tt.expectModel {
				t.Errorf("expected model %q to be requested, got %q", tt.expectModel, sent.Model)
			}
			if generated.Model != tt.expectModel {
				t.Errorf("expected model %q to be reported, got %q", tt.expectModel, generated.Model)
			}
		})
	}
}

func TestGenerate_SendsSystemAndUserPrompt(t *testing.T) {
	var sent openai.ChatCompletionRequest
	frames := append(contentFrames("TITEL: T\nText.\n"), usageFrame(10), "[DONE]")
	server := sseServer(t, frames, &sent)

	_, err := NewGenerator(testConfig(server.URL)).Generate(
		context.Background(),
		prompt.StoryRequest{Thema: "Freundschaft", Laenge: 5, Klassenstufe: "12"},
		StreamCallbacks{OnTitle: func(string) {}, OnChunk: func(string) {}},
	)
	if err != nil {
		t.Fatalf("expected the generation to succeed, got %v", err)
	}

	if len(sent.Messages) != 2 {
		t.Fatalf("expected a system and a user message, got %d", len(sent.Messages))
	}
	if sent.Messages[0].Role != openai.ChatMessageRoleSystem {
		t.Errorf("expected the first message to be the system prompt, got %q", sent.Messages[0].Role)
	}
	if !strings.Contains(sent.Messages[1].Content, "Freundschaft") {
		t.Error("expected the request parameters to reach the user prompt")
	}
	if sent.StreamOptions == nil || !sent.StreamOptions.IncludeUsage {
		t.Error("usage must be requested, otherwise token accounting stays at zero")
	}
}

func TestGenerate_UsesLastReportedUsage(t *testing.T) {
	// Providers may send usage more than once; the final value wins.
	frames := []string{usageFrame(100)}
	frames = append(frames, contentFrames("TITEL: T\nText.\n")...)
	frames = append(frames, usageFrame(2500), "[DONE]")
	server := sseServer(t, frames, nil)

	generated, err := NewGenerator(testConfig(server.URL)).Generate(
		context.Background(),
		prompt.StoryRequest{Thema: "Mut", Laenge: 5, Klassenstufe: "12"},
		StreamCallbacks{OnTitle: func(string) {}, OnChunk: func(string) {}},
	)
	if err != nil {
		t.Fatalf("expected the generation to succeed, got %v", err)
	}
	if generated.TokensUsed != 2500 {
		t.Errorf("expected the last reported usage (2500), got %d", generated.TokensUsed)
	}
}

func TestGenerate_SurvivesFramesWithoutChoices(t *testing.T) {
	// A keep-alive style frame carrying neither choices nor usage must be
	// skipped instead of panicking on an empty Choices slice.
	empty, _ := json.Marshal(map[string]any{"choices": []map[string]any{}})
	frames := []string{string(empty)}
	frames = append(frames, contentFrames("TITEL: Titel\nEin Text.\n")...)
	frames = append(frames, string(empty), usageFrame(42), "[DONE]")
	server := sseServer(t, frames, nil)

	generated, err := NewGenerator(testConfig(server.URL)).Generate(
		context.Background(),
		prompt.StoryRequest{Thema: "Mut", Laenge: 5, Klassenstufe: "12"},
		StreamCallbacks{OnTitle: func(string) {}, OnChunk: func(string) {}},
	)
	if err != nil {
		t.Fatalf("expected the generation to succeed, got %v", err)
	}
	if generated.Title != "Titel" || !strings.Contains(generated.Content, "Ein Text.") {
		t.Errorf("expected the story to survive the empty frames, got %+v", generated)
	}
}

func TestGenerate_FallsBackToDefaultTitleWhenStreamEndsEarly(t *testing.T) {
	// The stream ends before a TITEL marker ever arrives; finish() has to
	// resolve the title and replay the buffer as body text.
	frames := append(contentFrames("Eine Geschichte ganz ohne Marker."), usageFrame(10), "[DONE]")
	server := sseServer(t, frames, nil)

	var chunks []string
	generated, err := NewGenerator(testConfig(server.URL)).Generate(
		context.Background(),
		prompt.StoryRequest{Thema: "Mut", Laenge: 5, Klassenstufe: "12"},
		StreamCallbacks{OnTitle: func(string) {}, OnChunk: func(c string) { chunks = append(chunks, c) }},
	)
	if err != nil {
		t.Fatalf("expected the generation to succeed, got %v", err)
	}
	if generated.Title != "Ohne Titel" {
		t.Errorf("expected the fallback title, got %q", generated.Title)
	}
	if !strings.Contains(strings.Join(chunks, ""), "Eine Geschichte ganz ohne Marker.") {
		t.Errorf("expected the buffered text to be replayed as body, got %v", chunks)
	}
}

func TestGenerate_ReturnsErrorWhenStreamCannotBeOpened(t *testing.T) {
	origDelay := streamRetryDelay
	streamRetryDelay = time.Millisecond
	defer func() { streamRetryDelay = origDelay }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := NewGenerator(testConfig(server.URL)).Generate(
		context.Background(),
		prompt.StoryRequest{Thema: "Mut", Laenge: 5, Klassenstufe: "12"},
		StreamCallbacks{OnTitle: func(string) {}, OnChunk: func(string) {}},
	)
	if err == nil {
		t.Fatal("expected an error when the provider never opens the stream")
	}
	if !strings.Contains(err.Error(), "API request failed") {
		t.Errorf("expected an API request error, got %v", err)
	}
}

func TestGenerate_ReturnsErrorWhenStreamBreaksMidway(t *testing.T) {
	// A malformed frame after the stream is already open must surface as an
	// error rather than being silently treated as the end of the story.
	frames := append(contentFrames("TITEL: T\nErster Satz.\n"), "{not-valid-json")
	server := sseServer(t, frames, nil)

	_, err := NewGenerator(testConfig(server.URL)).Generate(
		context.Background(),
		prompt.StoryRequest{Thema: "Mut", Laenge: 5, Klassenstufe: "12"},
		StreamCallbacks{OnTitle: func(string) {}, OnChunk: func(string) {}},
	)
	if err == nil {
		t.Fatal("expected an error when the stream breaks midway")
	}
	if !strings.Contains(err.Error(), "stream interrupted") {
		t.Errorf("expected a stream interruption error, got %v", err)
	}
}

func TestGenerate_RespectsCancelledContext(t *testing.T) {
	server := sseServer(t, append(contentFrames("TITEL: T\nText.\n"), "[DONE]"), nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewGenerator(testConfig(server.URL)).Generate(
		ctx,
		prompt.StoryRequest{Thema: "Mut", Laenge: 5, Klassenstufe: "12"},
		StreamCallbacks{OnTitle: func(string) {}, OnChunk: func(string) {}},
	)
	if err == nil {
		t.Fatal("expected an error for an already cancelled context")
	}
}

func TestNewGenerator(t *testing.T) {
	// Setup
	cfg := &config.Config{
		OpenAIAPIKey:  "test-key",
		OpenAIBaseURL: "https://api.openai.com/v1",
		DefaultModel:  "gpt-4",
		AIProvider:    "openai",
	}

	// Execute
	generator := NewGenerator(cfg)

	// Assert - nil check with return
	if generator == nil {
		t.Fatal("Expected generator to be created")
		return
	}

	// Check config
	if generator.config.OpenAIAPIKey != cfg.OpenAIAPIKey {
		t.Error("Generator config doesn't match input config")
	}

	// Check GWS dictionary
	if generator.gwsDict == nil {
		t.Error("GWS dictionary should be initialized")
	}
	if len(generator.gwsDict) == 0 {
		t.Error("GWS dictionary should not be empty")
	}
}

func TestStory_Structure(t *testing.T) {
	// Setup
	story := &Story{
		Title:           "Der Hund",
		Content:         "Es war einmal...",
		Grundwortschatz: []string{"Hund", "Haus"},
		Model:           "gpt-4",
		Provider:        "openai",
		TokensUsed:      150,
		GenerationTime:  2.5,
	}

	// Assert
	if story.Title != "Der Hund" {
		t.Errorf("Expected Title 'Der Hund', got '%s'", story.Title)
	}
	if story.Content != "Es war einmal..." {
		t.Errorf("Expected Content 'Es war einmal...', got '%s'", story.Content)
	}
	if len(story.Grundwortschatz) != 2 {
		t.Errorf("Expected 2 GWS words, got %d", len(story.Grundwortschatz))
	}
	if story.Model != "gpt-4" {
		t.Errorf("Expected Model 'gpt-4', got '%s'", story.Model)
	}
	if story.Provider != "openai" {
		t.Errorf("Expected Provider 'openai', got '%s'", story.Provider)
	}
	if story.TokensUsed != 150 {
		t.Errorf("Expected TokensUsed 150, got %d", story.TokensUsed)
	}
	if story.GenerationTime != 2.5 {
		t.Errorf("Expected GenerationTime 2.5, got %f", story.GenerationTime)
	}
}

package story

import (
	"context"
	"fmt"
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

// Generator handles story generation
type Generator struct {
	config *config.Config
	gwsDict map[string]string
}

// NewGenerator creates a new story generator
func NewGenerator(cfg *config.Config) *Generator {
	return &Generator{
		config:  cfg,
		gwsDict: analysis.ExtractGrundwortschatzWords(),
	}
}

// Generate creates a story based on the given request
func (g *Generator) Generate(ctx context.Context, req prompt.StoryRequest) (*Story, error) {
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
	
	// Make API request with high token limit
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0.8,
		MaxTokens:   8000,
	})
	
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from API")
	}
	
	content := resp.Choices[0].Message.Content
	tokensUsed := resp.Usage.TotalTokens
	
	fmt.Printf("API Response - Tokens: %d, Zeichen: %d\n", tokensUsed, len(content))
	
	// Remove markdown formatting
	content = removeMarkdownFormatting(content)
	
	// Parse title and story
	title, storyText := parseStory(content)
	
	// Format and clean up ENDE marker
	storyText = formatEndeMarker(storyText)
	
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

func parseStory(content string) (string, string) {
	title := "Ohne Titel"
	story := content
	
	// Check for title in various formats (case-insensitive)
	contentUpper := strings.ToUpper(content)
	
	if strings.Contains(contentUpper, "TITEL:") {
		// Find the actual position in original content
		idx := strings.Index(contentUpper, "TITEL:")
		if idx >= 0 {
			// Get the part after "TITEL:" (or "Titel:" or "titel:")
			rest := strings.TrimSpace(content[idx+6:]) // 6 = len("TITEL:")
			titleEnd := strings.Index(rest, "\n")
			if titleEnd > 0 {
				title = strings.TrimSpace(rest[:titleEnd])
				story = strings.TrimSpace(rest[titleEnd+1:])
			} else {
				title = rest
				story = ""
			}
		}
	}
	
	// Note: Markdown formatting is already removed before this function
	
	return title, story
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

func formatEndeMarker(text string) string {
	// Find "ENDE" as a whole word (case-insensitive) and cut everything after it
	// Use regex to match ENDE as a complete word with word boundaries
	re := regexp.MustCompile(`(?i)\bENDE\b`)
	loc := re.FindStringIndex(text)
	
	if loc == nil {
		// No ENDE found, return as is
		return text
	}
	
	// Cut everything after "ENDE" (including the word itself)
	textBeforeEnde := strings.TrimSpace(text[:loc[0]])
	
	// Format ENDE marker centered with decorative line
	endeFormatted := "\n\n" + strings.Repeat(" ", 25) + " ★ ENDE ★ " + strings.Repeat(" ", 25)
	
	return textBeforeEnde + endeFormatted
}

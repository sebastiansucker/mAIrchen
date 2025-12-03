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
	
	// Use configured model if not specified in request
	model := req.Model
	if model == "" {
		model = g.config.DefaultModel
	}
	
	// Build prompts
	systemPrompt, userPrompt := prompt.BuildPrompt(req)
	
	// Create OpenAI client
	clientConfig := openai.DefaultConfig(g.config.OpenAIAPIKey)
	if g.config.OpenAIBaseURL != "" {
		clientConfig.BaseURL = g.config.OpenAIBaseURL
	}
	client := openai.NewClientWithConfig(clientConfig)
	
	// Estimate required tokens
	estimatedTokens := int(float64(req.Laenge) * 100 * 1.3) + 200
	
	// Make API request
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0.8,
		MaxTokens:   estimatedTokens,
	})
	
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from API")
	}
	
	content := resp.Choices[0].Message.Content
	
	// Parse title and story
	title, storyText := parseStory(content)
	
	// Find Grundwortschatz words
	gwsWords := analysis.FindGrundwortschatzInText(storyText, g.gwsDict)
	
	generationTime := time.Since(startTime).Seconds()
	
	return &Story{
		Title:           title,
		Content:         storyText,
		Grundwortschatz: gwsWords,
		Model:           model,
		Provider:        g.config.AIProvider,
		TokensUsed:      resp.Usage.TotalTokens,
		GenerationTime:  generationTime,
	}, nil
}

func parseStory(content string) (string, string) {
	title := "Ohne Titel"
	story := content
	
	if strings.Contains(content, "TITEL:") {
		parts := strings.SplitN(content, "TITEL:", 2)
		if len(parts) > 1 {
			rest := strings.TrimSpace(parts[1])
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
	
	// Remove markdown formatting
	story = removeMarkdownFormatting(story)
	
	return title, story
}

func removeMarkdownFormatting(text string) string {
	// Remove bold markers
	re := regexp.MustCompile(`\*\*(.*?)\*\*`)
	text = re.ReplaceAllString(text, "$1")
	
	// Remove italic markers
	re = regexp.MustCompile(`\*(.*?)\*`)
	text = re.ReplaceAllString(text, "$1")
	
	return text
}

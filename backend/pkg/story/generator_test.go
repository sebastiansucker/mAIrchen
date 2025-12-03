package story

import (
	"testing"
	"github.com/sebastiansucker/mAIrchen/backend/pkg/config"
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
			expectStory: "Der Fuchs war sehr schlau.",
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

	// Assert
	if generator == nil {
		t.Error("Expected generator to be created")
	}
	if generator.config.OpenAIAPIKey != cfg.OpenAIAPIKey {
		t.Error("Generator config doesn't match input config")
	}
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
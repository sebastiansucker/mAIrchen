package story

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
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
	if strings.Contains(got, "ENDE") {
		t.Errorf("raw ENDE marker must never be emitted as a chunk, got %q", got)
	}
	if !strings.Contains(p.fullStory.String(), "★ ENDE ★") {
		t.Errorf("expected decorative ENDE footer in full story, got %q", p.fullStory.String())
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
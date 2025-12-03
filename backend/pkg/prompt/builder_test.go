package prompt

import (
	"strings"
	"testing"
)

func TestBuildPrompt_Klasse12(t *testing.T) {
	// Setup
	req := StoryRequest{
		Thema:          "Freundschaft",
		PersonenTiere:  "Ein kleiner Hase",
		Ort:            "im Wald",
		Stimmung:       "fröhlich",
		Laenge:         2,
		Klassenstufe:   "12",
		Stil:           "",
	}

	// Execute
	systemPrompt, userPrompt := BuildPrompt(req)

	// Assert
	if !strings.Contains(systemPrompt, "Kinder der Klassenstufen 1 & 2") {
		t.Error("System prompt should mention Klasse 1 & 2")
	}

	if !strings.Contains(userPrompt, "120-140 Wörter") {
		t.Error("User prompt should mention 120-140 words for 2 minutes, Klasse 1-2")
	}

	if !strings.Contains(userPrompt, "Freundschaft") {
		t.Error("User prompt should contain the theme")
	}

	if !strings.Contains(userPrompt, "Ein kleiner Hase") {
		t.Error("User prompt should contain the characters")
	}

	if !strings.Contains(userPrompt, "im Wald") {
		t.Error("User prompt should contain the location")
	}

	if !strings.Contains(userPrompt, "fröhlich") {
		t.Error("User prompt should contain the mood")
	}

	if !strings.Contains(userPrompt, "sehr einfach") {
		t.Error("User prompt should mention difficulty for Klasse 1-2")
	}

	if !strings.Contains(userPrompt, "TITEL:") {
		t.Error("User prompt should contain format instructions with TITEL:")
	}
}

func TestBuildPrompt_Klasse34(t *testing.T) {
	// Setup
	req := StoryRequest{
		Thema:          "Abenteuer",
		PersonenTiere:  "Eine mutige Maus",
		Ort:            "in einer alten Mühle",
		Stimmung:       "spannend",
		Laenge:         3,
		Klassenstufe:   "34",
		Stil:           "",
	}

	// Execute
	systemPrompt, userPrompt := BuildPrompt(req)

	// Assert
	if !strings.Contains(systemPrompt, "Kinder der Klassenstufen 3 & 4") {
		t.Error("System prompt should mention Klasse 3 & 4")
	}

	if !strings.Contains(userPrompt, "240-300 Wörter") {
		t.Error("User prompt should mention 240-300 words for 3 minutes, Klasse 3-4")
	}

	if !strings.Contains(userPrompt, "kindgerecht mit etwas längeren Sätzen") {
		t.Error("User prompt should mention difficulty for Klasse 3-4")
	}
}

func TestBuildPrompt_WithStil(t *testing.T) {
	// Setup
	req := StoryRequest{
		Thema:          "Magie",
		PersonenTiere:  "Eine Hexe",
		Ort:            "im Zauberwald",
		Stimmung:       "mysteriös",
		Laenge:         2,
		Klassenstufe:   "34",
		Stil:           "Michael Ende",
	}

	// Execute
	_, userPrompt := BuildPrompt(req)

	// Assert
	if !strings.Contains(userPrompt, "Stil/Genre: Michael Ende") {
		t.Error("User prompt should contain the style when provided")
	}
}

func TestBuildPrompt_WithoutStil(t *testing.T) {
	// Setup
	req := StoryRequest{
		Thema:          "Magie",
		PersonenTiere:  "Eine Hexe",
		Ort:            "im Zauberwald",
		Stimmung:       "mysteriös",
		Laenge:         2,
		Klassenstufe:   "34",
		Stil:           "",
	}

	// Execute
	_, userPrompt := BuildPrompt(req)

	// Assert
	if strings.Contains(userPrompt, "Stil/Genre:") {
		t.Error("User prompt should not contain style instruction when not provided")
	}
}

func TestGetGWSContent(t *testing.T) {
	// Execute
	content := GetGWSContent()

	// Assert
	if len(content) == 0 {
		t.Error("Expected GWS content to be non-empty")
	}

	if !strings.Contains(content, "Grundwortschatz") {
		t.Error("Expected GWS content to contain 'Grundwortschatz'")
	}
}

func TestSplitGWSContent(t *testing.T) {
	// Execute
	parts := splitGWSContent()

	// Assert
	if len(parts) == 0 {
		t.Error("Expected at least one part from split")
	}

	// If separator exists, should have 2 parts
	if len(parts) == 2 {
		if !strings.Contains(parts[1], "Grundwortschatz für Jahrgangsstufen 3 und 4") {
			t.Error("Second part should contain Klasse 3-4 section")
		}
	}
}

func TestBuildPrompt_WordCountCalculation(t *testing.T) {
	tests := []struct {
		klassenstufe string
		laenge       int
		expectedMin  string
		expectedMax  string
	}{
		{"12", 1, "60-70", "70"},
		{"12", 3, "180-210", "210"},
		{"34", 1, "80-100", "100"},
		{"34", 5, "400-500", "500"},
	}

	for _, tt := range tests {
		t.Run(tt.klassenstufe+"_"+string(rune(tt.laenge+'0')), func(t *testing.T) {
			req := StoryRequest{
				Thema:          "Test",
				PersonenTiere:  "Test",
				Ort:            "Test",
				Stimmung:       "Test",
				Laenge:         tt.laenge,
				Klassenstufe:   tt.klassenstufe,
			}

			_, userPrompt := BuildPrompt(req)

			if !strings.Contains(userPrompt, tt.expectedMin+" Wörter") {
				t.Errorf("Expected prompt to contain '%s Wörter'", tt.expectedMin)
			}

			if !strings.Contains(userPrompt, tt.expectedMax+" Wörtern") {
				t.Errorf("Expected prompt to contain '%s Wörtern'", tt.expectedMax)
			}
		})
	}
}

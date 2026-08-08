package prompt

import (
	"strings"
	"testing"

	"github.com/sebastiansucker/mAIrchen/backend/pkg/data"
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

	if !strings.Contains(userPrompt, "100-180") {
		t.Error("User prompt should mention 100-180 words range for 2 minutes, Klasse 1-2")
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

	if strings.Contains(userPrompt, "Grundwortschatz für Jahrgangsstufen 3 und 4") {
		t.Error("User prompt for Klasse 1-2 should not contain the Klasse 3-4 Grundwortschatz section")
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

	if !strings.Contains(userPrompt, "240-360") {
		t.Error("User prompt should mention 240-360 words range for 3 minutes, Klasse 3-4")
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

func TestKlasse12Grundwortschatz(t *testing.T) {
	// Execute
	content := klasse12Grundwortschatz()

	// Assert
	if len(content) == 0 {
		t.Error("Expected Klasse 1-2 Grundwortschatz content to be non-empty")
	}

	if strings.Contains(content, "Grundwortschatz für Jahrgangsstufen 3 und 4") {
		t.Error("Klasse 1-2 content should not contain the Klasse 3-4 section")
	}

	if len(content) >= len(data.GrundwortschatzContent) {
		t.Error("Klasse 1-2 content should be a strict subset of the full Grundwortschatz content")
	}

	// Calling again must return the identical cached value.
	if again := klasse12Grundwortschatz(); again != content {
		t.Error("Expected klasse12Grundwortschatz to return the cached value on repeated calls")
	}
}

func TestBuildPrompt_WordCountCalculation(t *testing.T) {
	tests := []struct {
		klassenstufe string
		laenge       int
		expectedMin  string
		expectedMax  string
	}{
		{"12", 1, "50-90", "90"},
		{"12", 3, "150-270", "270"},
		{"34", 1, "80-120", "120"},
		{"34", 5, "400-600", "600"},
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

			// Die Wortanzahl wird jetzt nicht mehr direkt im ersten Teil erwähnt, nur die Orientierung
			if !strings.Contains(userPrompt, tt.expectedMin) {
				t.Errorf("Expected prompt to contain '%s'", tt.expectedMin)
			}
		})
	}
}

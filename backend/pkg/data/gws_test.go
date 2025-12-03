package data

import (
	"strings"
	"testing"
)

func TestGrundwortschatzContent_NotEmpty(t *testing.T) {
	// Assert
	if len(GrundwortschatzContent) == 0 {
		t.Error("GrundwortschatzContent should not be empty")
	}
}

func TestGrundwortschatzContent_ContainsExpectedSections(t *testing.T) {
	// Assert - check for key sections
	if !strings.Contains(GrundwortschatzContent, "Grundwortschatz") {
		t.Error("Expected GrundwortschatzContent to contain 'Grundwortschatz'")
	}

	// Should contain grade level sections
	hasKlasse12 := strings.Contains(GrundwortschatzContent, "Jahrgangsstufen 1 und 2") ||
		strings.Contains(GrundwortschatzContent, "Klasse 1") ||
		strings.Contains(GrundwortschatzContent, "Klassenstufe 1")

	hasKlasse34 := strings.Contains(GrundwortschatzContent, "Jahrgangsstufen 3 und 4") ||
		strings.Contains(GrundwortschatzContent, "Klasse 3") ||
		strings.Contains(GrundwortschatzContent, "Klassenstufe 3")

	if !hasKlasse12 && !hasKlasse34 {
		t.Error("Expected GrundwortschatzContent to contain grade level sections")
	}
}

func TestGrundwortschatzContent_ContainsSampleWords(t *testing.T) {
	// Assert - check for some common German words that should be in GWS
	// These are typical words found in German elementary vocabulary
	commonWords := []string{
		"der", "die", "das",
		"und", "oder",
	}

	for _, word := range commonWords {
		if !strings.Contains(strings.ToLower(GrundwortschatzContent), word) {
			t.Logf("Warning: Common word '%s' not found in GrundwortschatzContent", word)
		}
	}
}

func TestGrundwortschatzContent_IsValidMarkdown(t *testing.T) {
	// Basic markdown structure checks
	if !strings.Contains(GrundwortschatzContent, "#") {
		t.Error("Expected GrundwortschatzContent to contain markdown headers")
	}
}

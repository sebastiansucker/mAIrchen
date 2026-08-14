package analysis

import (
	"reflect"
	"strings"
	"testing"
)

func TestExtractGrundwortschatzWords(t *testing.T) {
	// Execute
	words := ExtractGrundwortschatzWords()

	// Assert
	if len(words) == 0 {
		t.Error("Expected words to be extracted, got empty map")
	}

	// The Grundwortschatz file has ~599 entries; article-prefixed lines
	// (e.g. "- der Hund (Hunde)") must not collapse into a single "der"/"die"/"das" key.
	if len(words) < 500 {
		t.Errorf("Expected close to 599 extracted words, got %d (article-prefixed nouns may be lost)", len(words))
	}

	// Nouns with an article prefix must resolve to the noun, not the article.
	nounCases := map[string]string{
		"hund":  "Hund",
		"katze": "Katze",
		"auto":  "Auto",
	}
	for lower, expected := range nounCases {
		word, exists := words[lower]
		if !exists {
			t.Errorf("Expected key %q to exist in extracted words", lower)
			continue
		}
		if word != expected {
			t.Errorf("Expected %q, got %q", expected, word)
		}
	}

	// "der"/"die"/"das" themselves must not appear as extracted words.
	for _, article := range []string{"der", "die", "das"} {
		if word, exists := words[article]; exists {
			t.Errorf("Did not expect article %q to be extracted as a word, got %q", article, word)
		}
	}
}

func TestFindGrundwortschatzInText(t *testing.T) {
	// Setup - create a simple test dictionary
	gwsDict := map[string]string{
		"hund":   "Hund",
		"katze":  "Katze",
		"haus":   "Haus",
		"baum":   "Baum",
		"sonne":  "Sonne",
	}

	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "Single word match",
			text:     "Der Hund bellt laut.",
			expected: []string{"Hund"},
		},
		{
			name:     "Multiple word matches",
			text:     "Der Hund und die Katze spielen im Haus.",
			expected: []string{"Haus", "Hund", "Katze"},
		},
		{
			name:     "Case insensitive match",
			text:     "Die SONNE scheint und der hund spielt.",
			expected: []string{"Hund", "Sonne"},
		},
		{
			name:     "No matches",
			text:     "Der Vogel fliegt.",
			expected: []string{},
		},
		{
			name:     "Exact word matches only",
			text:     "Der Hund und die Katze spielen zusammen.",
			expected: []string{"Hund", "Katze"},
		},
		{
			name:     "Empty text",
			text:     "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			result := FindGrundwortschatzInText(tt.text, gwsDict)

			// Assert
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFindGrundwortschatzInText_Sorting(t *testing.T) {
	// Setup
	gwsDict := map[string]string{
		"zebra":  "Zebra",
		"apfel":  "Apfel",
		"maus":   "Maus",
	}
	text := "Die Zebra, Maus und Apfel sind hier."

	// Execute
	result := FindGrundwortschatzInText(text, gwsDict)

	// Assert - should be sorted alphabetically
	expected := []string{"Apfel", "Maus", "Zebra"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected sorted %v, got %v", expected, result)
	}
}

func TestFindGrundwortschatzInText_UmlautWordStart(t *testing.T) {
	// Setup - words starting with an umlaut, as found in the embedded
	// Grundwortschatz (e.g. "über", "ähnlich", "ändern", "ärgern").
	gwsDict := map[string]string{
		"über":    "Über",
		"ähnlich": "Ähnlich",
		"ändern":  "Ändern",
		"ärgern":  "Ärgern",
	}

	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "umlaut word after whitespace",
			text:     "Der Hase sprang über den Zaun.",
			expected: []string{"Über"},
		},
		{
			name:     "umlaut word at sentence start",
			text:     "Ähnlich sah es gestern schon aus.",
			expected: []string{"Ähnlich"},
		},
		{
			name:     "multiple umlaut words",
			text:     "Er wollte nichts ändern, doch das würde ihn nur ärgern.",
			expected: []string{"Ändern", "Ärgern"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindGrundwortschatzInText(tt.text, gwsDict)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFindGrundwortschatzInText_NoDuplicates(t *testing.T) {
	// Setup
	gwsDict := map[string]string{
		"hund": "Hund",
	}
	text := "Der Hund läuft. Der Hund bellt. Die Hunde spielen."

	// Execute
	result := FindGrundwortschatzInText(text, gwsDict)

	// Assert - should appear only once
	expected := []string{"Hund"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

// BenchmarkFindGrundwortschatzInText measures FindGrundwortschatzInText against
// the full embedded Grundwortschatz dictionary and a realistic story-sized text,
// guarding against regressions like comparing every token against every
// dictionary entry instead of looking up prefixes directly (see issue #46).
func BenchmarkFindGrundwortschatzInText(b *testing.B) {
	gwsDict := ExtractGrundwortschatzWords()
	text := strings.Repeat(
		"Der kleine Hund lief mit der Katze durch den Wald und über die Wiese zum Haus, "+
			"wo die Sonne schien und ein Vogel fröhlich sang, während Mutter und Vater "+
			"gemeinsam ein Buch über einen mutigen Bären lasen. ",
		20,
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FindGrundwortschatzInText(text, gwsDict)
	}
}

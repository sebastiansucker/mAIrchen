package analysis

import (
	"reflect"
	"testing"
)

func TestExtractGrundwortschatzWords(t *testing.T) {
	// Execute
	words := ExtractGrundwortschatzWords()

	// Assert
	if len(words) == 0 {
		t.Error("Expected words to be extracted, got empty map")
	}

	// Check that lowercase keys map to correctly capitalized values
	if word, exists := words["hund"]; exists {
		if word != "Hund" {
			t.Errorf("Expected 'Hund', got '%s'", word)
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

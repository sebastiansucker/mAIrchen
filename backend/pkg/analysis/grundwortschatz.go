package analysis

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/sebastiansucker/mAIrchen/backend/pkg/data"
)

// ExtractGrundwortschatzWords extracts words from the Grundwortschatz file
// Returns a map where keys are lowercase words and values are the correctly capitalized versions
func ExtractGrundwortschatzWords() map[string]string {
	gwsDict := make(map[string]string)
	re := regexp.MustCompile(`(?m)^\s*-\s+(?:(?:der|die|das)\s+)?(\S+)`)

	lines := strings.Split(data.GrundwortschatzContent, "\n")
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			word := matches[1]
			gwsDict[strings.ToLower(word)] = word
		}
	}
	
	return gwsDict
}

// FindGrundwortschatzInText finds Grundwortschatz words in the given text.
// A text word counts as a match if it starts with a Grundwortschatz word
// (e.g. "Hunde" matches the dictionary entry "hund"). For each token, this
// looks up successive prefixes directly in gwsDict instead of comparing the
// token against every dictionary entry, so cost scales with token length
// rather than dictionary size.
// Returns a sorted list of words with correct capitalization
func FindGrundwortschatzInText(text string, gwsDict map[string]string) []string {
	foundWords := make(map[string]bool)

	checked := make(map[string]bool)
	for _, token := range extractWordTokens(text) {
		lowerToken := strings.ToLower(token)
		if checked[lowerToken] {
			continue
		}
		checked[lowerToken] = true

		runes := []rune(lowerToken)
		for i := 1; i <= len(runes); i++ {
			if correctWord, ok := gwsDict[string(runes[:i])]; ok {
				foundWords[correctWord] = true
			}
		}
	}

	result := make([]string, 0, len(foundWords))
	for word := range foundWords {
		result = append(result, word)
	}
	sort.Strings(result)

	return result
}

// extractWordTokens splits text into runs of Unicode letters, so German
// umlauts (ä, ö, ü) and ß are treated as word characters instead of the
// ASCII-only definition used by regexp's \b/\w.
func extractWordTokens(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
}

package analysis

import (
	"regexp"
	"sort"
	"strings"

	"github.com/sebastiansucker/mAIrchen/backend/pkg/data"
)

// ExtractGrundwortschatzWords extracts words from the Grundwortschatz file
// Returns a map where keys are lowercase words and values are the correctly capitalized versions
func ExtractGrundwortschatzWords() map[string]string {
	gwsDict := make(map[string]string)
	re := regexp.MustCompile(`(?m)^\s*-\s+(\S+)`)
	
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

// FindGrundwortschatzInText finds Grundwortschatz words in the given text
// Returns a sorted list of words with correct capitalization
func FindGrundwortschatzInText(text string, gwsDict map[string]string) []string {
	foundWords := make(map[string]bool)
	textLower := strings.ToLower(text)
	
	for lowerWord, correctWord := range gwsDict {
		pattern := `\b` + regexp.QuoteMeta(lowerWord) + `\w*\b`
		re := regexp.MustCompile(pattern)
		if re.MatchString(textLower) {
			foundWords[correctWord] = true
		}
	}
	
	result := make([]string, 0, len(foundWords))
	for word := range foundWords {
		result = append(result, word)
	}
	sort.Strings(result)
	
	return result
}

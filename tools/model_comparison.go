package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/sebastiansucker/mAIrchen/backend/pkg/analysis"
	"github.com/sebastiansucker/mAIrchen/backend/pkg/config"
	"github.com/sebastiansucker/mAIrchen/backend/pkg/prompt"
	"github.com/sebastiansucker/mAIrchen/backend/pkg/story"
)

// ModelConfig holds configuration for a specific model
type ModelConfig struct {
	Name     string
	Provider string
	Model    string
	APIKey   string
	BaseURL  string
}

// TestCase represents a test scenario
type TestCase struct {
	Name          string
	Thema         string
	PersonenTiere string
	Ort           string
	Stimmung      string
	Laenge        int
	Klassenstufe  string
	Stil          string
}

// TestResult holds the results of a single test
type TestResult struct {
	TestCase        string                  `json:"test_case"`
	Success         bool                    `json:"success"`
	Provider        string                  `json:"provider"`
	Model           string                  `json:"model"`
	GenerationTime  float64                 `json:"generation_time"`
	Title           string                  `json:"title,omitempty"`
	WordCount       int                     `json:"word_count,omitempty"`
	ParagraphCount  int                     `json:"paragraph_count,omitempty"`
	DialogueCount   int                     `json:"dialogue_count,omitempty"`
	Grundwortschatz GrundwortschatzAnalysis `json:"grundwortschatz,omitempty"`
	Quality         QualityAssessment       `json:"quality,omitempty"`
	TokensUsed      int                     `json:"tokens_used,omitempty"`
	StoryPreview    string                  `json:"story_preview,omitempty"`
	SystemPrompt    string                  `json:"system_prompt,omitempty"`
	UserPrompt      string                  `json:"user_prompt,omitempty"`
	Error           string                  `json:"error,omitempty"`
}

// GrundwortschatzAnalysis holds GWS analysis results
type GrundwortschatzAnalysis struct {
	UniqueWords      int      `json:"unique_words"`
	TotalOccurrences int      `json:"total_occurrences"`
	Percentage       float64  `json:"percentage"`
	TopWords         []string `json:"top_words"`
}

// QualityAssessment holds story quality metrics
type QualityAssessment struct {
	HasProperEnding    bool     `json:"has_proper_ending"`
	HasEndeMarker      bool     `json:"has_ende_marker"`
	IsComplete         bool     `json:"is_complete"`
	HasClearStructure  bool     `json:"has_clear_structure"`
	HasDialogue        bool     `json:"has_dialogue"`
	EndingIndicators   []string `json:"ending_indicators,omitempty"`
	IssuesFound        []string `json:"issues_found,omitempty"`
	QualityScore       float64  `json:"quality_score"` // 0-100
}

// ModelResults holds all results for a model
type ModelResults struct {
	Model     string       `json:"model"`
	Provider  string       `json:"provider"`
	BaseURL   string       `json:"base_url"`
	Timestamp string       `json:"timestamp"`
	Tests     []TestResult `json:"tests"`
}

var testCases = []TestCase{
	{
		Name:          "Klasse 1-2: Kurz (5 Min) - Tiere",
		Thema:         "Tiere und Natur",
		PersonenTiere: "Ein kleiner Hase",
		Ort:           "auf der Wiese",
		Stimmung:      "fröhlich",
		Laenge:        5,
		Klassenstufe:  "12",
	},
	{
		Name:          "Klasse 1-2: Standard (10 Min) - Freundschaft",
		Thema:         "Freundschaft",
		PersonenTiere: "Ein Igel und ein Eichhörnchen",
		Ort:           "im Wald",
		Stimmung:      "herzlich",
		Laenge:        10,
		Klassenstufe:  "12",
	},
	{
		Name:          "Klasse 3-4: Kurz (5 Min) - Freundschaft",
		Thema:         "Freundschaft",
		PersonenTiere: "Ein kleiner Igel",
		Ort:           "im Wald",
		Stimmung:      "herzlich",
		Laenge:        5,
		Klassenstufe:  "34",
	},
	{
		Name:          "Klasse 3-4: Standard (10 Min) - Abenteuer",
		Thema:         "Abenteuer",
		PersonenTiere: "Eine mutige Maus",
		Ort:           "in einer alten Mühle",
		Stimmung:      "spannend",
		Laenge:        10,
		Klassenstufe:  "34",
	},
	{
		Name:          "Klasse 3-4: Lang (15 Min) - Zauber",
		Thema:         "Zauber und Magie",
		PersonenTiere: "Eine junge Hexe und ihr Kater",
		Ort:           "in einem verzauberten Garten",
		Stimmung:      "mysteriös",
		Laenge:        15,
		Klassenstufe:  "34",
	},
}

func main() {
	log.Println("🧪 mAIrchen - Modell-Vergleichstest (Go)")
	log.Println("========================================")

	// Load .env file if exists
	_ = godotenv.Load() // Ignore error if .env doesn't exist

	// Load model configurations from environment
	modelConfigs := loadModelConfigs()

	if len(modelConfigs) == 0 {
		log.Fatal("❌ Keine Modelle konfiguriert. Bitte .env Datei prüfen.")
	}

	log.Printf("📋 %d Modelle × %d Test-Cases = %d Tests\n\n",
		len(modelConfigs), len(testCases), len(modelConfigs)*len(testCases))

	// Run tests
	allResults := make([]ModelResults, 0)
	gwsDict := analysis.ExtractGrundwortschatzWords()

	for _, modelConfig := range modelConfigs {
		log.Printf("\n%s\n", strings.Repeat("=", 60))
		log.Printf("🤖 Modell: %s (%s)", modelConfig.Name, modelConfig.Provider)
		log.Printf("\n%s\n", strings.Repeat("=", 60))

		modelResults := ModelResults{
			Model:     modelConfig.Name,
			Provider:  modelConfig.Provider,
			BaseURL:   modelConfig.BaseURL,
			Timestamp: time.Now().Format(time.RFC3339),
			Tests:     make([]TestResult, 0),
		}

		// Create custom config for this model
		cfg := &config.Config{
			AIProvider:    modelConfig.Provider,
			OpenAIAPIKey:  modelConfig.APIKey,
			OpenAIBaseURL: modelConfig.BaseURL,
			DefaultModel:  modelConfig.Model,
		}
		gen := story.NewGenerator(cfg)

		for _, testCase := range testCases {
			log.Printf("  📝 Teste: %s", testCase.Name)

			result := runTest(gen, testCase, modelConfig, gwsDict)
			modelResults.Tests = append(modelResults.Tests, result)

			if result.Success {
				log.Printf("    ✅ %.1fs | %d Wörter | GWS: %d Wörter\n",
					result.GenerationTime, result.WordCount, result.Grundwortschatz.UniqueWords)
			} else {
				log.Printf("    ❌ Fehler: %s\n", result.Error)
			}

			// Small delay between requests
			time.Sleep(1 * time.Second)
		}

		allResults = append(allResults, modelResults)
		log.Println()
	}

	log.Printf("\n%s\n", strings.Repeat("=", 60))
	log.Println("✅ Alle Tests abgeschlossen!")
	log.Printf("%s\n\n", strings.Repeat("=", 60))

	// Print summary to terminal
	printSummary(allResults)

	// Save results
	saveResults(allResults)
	generateReport(allResults)
}

func loadModelConfigs() []ModelConfig {
	configs := make([]ModelConfig, 0)

	// Check for Ollama Cloud
	if apiKey := os.Getenv("OLLAMA_API_KEY"); apiKey != "" {
		models := os.Getenv("OLLAMA_MODELS")
		if models == "" {
			models = "ministral-3:8b-cloud"
		}
		for _, model := range strings.Split(models, ",") {
			model = strings.TrimSpace(model)
			if model != "" {
				configs = append(configs, ModelConfig{
					Name:     model,
					Provider: "ollama-cloud",
					Model:    model,
					APIKey:   apiKey,
					BaseURL:  "https://ollama.com/v1",
				})
			}
		}
	}

	// Check for Ollama Local
	if baseURL := os.Getenv("OLLAMA_BASE_URL"); baseURL != "" {
		models := os.Getenv("OLLAMA_LOCAL_MODELS")
		if models == "" {
			models = "mistral:7b"
		}
		for _, model := range strings.Split(models, ",") {
			model = strings.TrimSpace(model)
			if model != "" {
				configs = append(configs, ModelConfig{
					Name:     model,
					Provider: "ollama-local",
					Model:    model,
					APIKey:   "dummy-key",
					BaseURL:  baseURL,
				})
			}
		}
	}

	// Check for OpenAI/Mistral
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		baseURL := os.Getenv("OPENAI_BASE_URL")
		if baseURL == "" {
			baseURL = "https://api.mistral.ai/v1"
		}
		models := os.Getenv("MISTRAL_MODELS")
		if models == "" {
			models = "mistral-small-latest"
		}
		for _, model := range strings.Split(models, ",") {
			model = strings.TrimSpace(model)
			if model != "" {
				configs = append(configs, ModelConfig{
					Name:     model,
					Provider: "mistral-api",
					Model:    model,
					APIKey:   apiKey,
					BaseURL:  baseURL,
				})
			}
		}
	}

	return configs
}

func runTest(gen *story.Generator, testCase TestCase, modelConfig ModelConfig, gwsDict map[string]string) TestResult {
	req := prompt.StoryRequest{
		Thema:         testCase.Thema,
		PersonenTiere: testCase.PersonenTiere,
		Ort:           testCase.Ort,
		Stimmung:      testCase.Stimmung,
		Laenge:        testCase.Laenge,
		Klassenstufe:  testCase.Klassenstufe,
		Stil:          testCase.Stil,
		Model:         modelConfig.Model,
	}

	// Build prompts for logging
	systemPrompt, userPrompt := prompt.BuildPrompt(req)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	startTime := time.Now()
	generatedStory, err := gen.Generate(ctx, req, story.StreamCallbacks{
		OnTitle: func(string) {},
		OnChunk: func(string) {},
	})
	generationTime := time.Since(startTime).Seconds()

	if err != nil {
		return TestResult{
			TestCase:       testCase.Name,
			Success:        false,
			Provider:       modelConfig.Provider,
			Model:          modelConfig.Name,
			GenerationTime: generationTime,
			Error:          err.Error(),
		}
	}

	// Analyze story
	wordCount := countWords(generatedStory.Content)
	paragraphCount := countParagraphs(generatedStory.Content)
	dialogueCount := countDialogues(generatedStory.Content)

	gwsAnalysis := analyzeGrundwortschatz(generatedStory.Content, generatedStory.Grundwortschatz, gwsDict)
	qualityAssessment := assessQuality(generatedStory.Content, generatedStory.Title, paragraphCount)

	preview := generatedStory.Content
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}

	return TestResult{
		TestCase:        testCase.Name,
		Success:         true,
		Provider:        modelConfig.Provider,
		Model:           modelConfig.Name,
		GenerationTime:  generationTime,
		Title:           generatedStory.Title,
		WordCount:       wordCount,
		ParagraphCount:  paragraphCount,
		DialogueCount:   dialogueCount,
		Quality:         qualityAssessment,
		Grundwortschatz: gwsAnalysis,
		TokensUsed:      generatedStory.TokensUsed,
		StoryPreview:    preview,
		SystemPrompt:    systemPrompt,
		UserPrompt:      userPrompt,
	}
}

func countWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

func countParagraphs(text string) int {
	paragraphs := strings.Split(text, "\n\n")
	count := 0
	for _, p := range paragraphs {
		if strings.TrimSpace(p) != "" {
			count++
		}
	}
	return count
}

func countDialogues(text string) int {
	re := regexp.MustCompile(`[„"].*?["""]`)
	matches := re.FindAllString(text, -1)
	return len(matches)
}

func analyzeGrundwortschatz(text string, foundWords []string, gwsDict map[string]string) GrundwortschatzAnalysis {
	totalGWSWords := len(gwsDict)
	uniqueWords := len(foundWords)

	// Count occurrences
	textLower := strings.ToLower(text)
	totalOccurrences := 0
	for _, word := range foundWords {
		wordLower := strings.ToLower(word)
		pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(wordLower) + `\w*\b`)
		matches := pattern.FindAllString(textLower, -1)
		totalOccurrences += len(matches)
	}

	percentage := 0.0
	if totalGWSWords > 0 {
		percentage = float64(uniqueWords) / float64(totalGWSWords) * 100
	}

	// Top 5 words
	topWords := foundWords
	if len(topWords) > 5 {
		topWords = topWords[:5]
	}

	return GrundwortschatzAnalysis{
		UniqueWords:      uniqueWords,
		TotalOccurrences: totalOccurrences,
		Percentage:       percentage,
		TopWords:         topWords,
	}
}

func assessQuality(content string, title string, paragraphCount int) QualityAssessment {
	issues := []string{}
	endingIndicators := []string{}
	
	// Check for title
	if title == "" || strings.TrimSpace(title) == "" {
		issues = append(issues, "Kein Titel vorhanden")
	}
	
	// Check story length (minimum reasonable length)
	wordCount := countWords(content)
	if wordCount < 100 {
		issues = append(issues, fmt.Sprintf("Geschichte zu kurz (%d Wörter)", wordCount))
	}
	
	// Check for structure (paragraphs)
	hasStructure := paragraphCount >= 3
	if !hasStructure {
		issues = append(issues, fmt.Sprintf("Zu wenige Absätze für klare Struktur (%d)", paragraphCount))
	}
	
	// Check for dialogue
	hasDialogue := countDialogues(content) > 0
	
	// Check for proper ending indicators
	contentLower := strings.ToLower(content)
	lastPart := ""
	if len(content) > 200 {
		lastPart = contentLower[len(contentLower)-200:]
	} else {
		lastPart = contentLower
	}
	
	// Check if original response contained ENDE marker (before it was removed)
	hasEndeMarker := strings.Contains(contentLower, "ende\n") || 
		strings.HasSuffix(strings.TrimSpace(contentLower), "ende")
	
	if hasEndeMarker {
		endingIndicators = append(endingIndicators, "ENDE-Marker gefunden")
	} else {
		issues = append(issues, "Kein ENDE-Marker gefunden")
	}
	
	endingPhrases := []string{
		"ende", "schluss", "seitdem", "von da an", "von nun an",
		"für immer", "glücklich", "und so", "und wenn",
		"bis heute", "nie wieder", "von diesem tag an",
		"lebten sie", "waren sie", "blieb", "war es",
	}
	
	for _, phrase := range endingPhrases {
		if strings.Contains(lastPart, phrase) {
			endingIndicators = append(endingIndicators, phrase)
		}
	}
	
	// Check for abrupt ending (story ends mid-sentence or with incomplete thought)
	hasProperEnding := true
	trimmedContent := strings.TrimSpace(content)
	
	// Check if ends with proper punctuation
	if len(trimmedContent) > 0 {
		lastChar := trimmedContent[len(trimmedContent)-1]
		if lastChar != '.' && lastChar != '!' && lastChar != '?' {
			issues = append(issues, "Geschichte endet ohne Satzzeichen")
			hasProperEnding = false
		}
	}
	
	// Check if ends with dialogue (often incomplete)
	if strings.HasSuffix(trimmedContent, "\"") || strings.HasSuffix(trimmedContent, string(rune(0x201C))) || strings.HasSuffix(trimmedContent, string(rune(0x201D))) {
		if len(endingIndicators) == 0 {
			issues = append(issues, "Geschichte endet möglicherweise mit Dialog statt Abschluss")
			hasProperEnding = false
		}
	}
	
	// Check for incomplete sentences at the end
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) > 0 {
		lastLine := strings.TrimSpace(lines[len(lines)-1])
		lastLineLower := strings.ToLower(lastLine)
		
		// Allow typical fairy tale endings that start with "und so", "und wenn", etc.
		validEndingStarts := []string{"und so", "und wenn", "und seitdem", "und von da an"}
		isValidEnding := false
		for _, validStart := range validEndingStarts {
			if strings.HasPrefix(lastLineLower, validStart) {
				isValidEnding = true
				break
			}
		}
		
		// Only flag as incomplete if it starts with problematic words AND is not a valid ending
		if !isValidEnding {
			incompleteWords := []string{"und", "aber", "doch", "da", "als", "wenn", "weil", "dass"}
			for _, word := range incompleteWords {
				if strings.HasPrefix(lastLineLower, word+" ") {
					issues = append(issues, fmt.Sprintf("Letzter Satz beginnt mit '%s' (möglicherweise unvollständig)", word))
					hasProperEnding = false
					break
				}
			}
		}
	}
	
	// Determine if story is complete
	// Story is complete if: has ENDE marker OR (has proper ending AND ending indicators AND good structure)
	isComplete := hasEndeMarker || (hasProperEnding && len(endingIndicators) > 1 && paragraphCount >= 3)
	
	// Calculate quality score (0-100)
	score := 100.0
	if !hasEndeMarker {
		score -= 30 // Heavy penalty for missing ENDE marker
		issues = append(issues, "ENDE-Marker fehlt")
	}
	if !hasProperEnding {
		score -= 20
	}
	if len(endingIndicators) <= 1 { // Only ENDE marker or nothing
		score -= 15
	}
	if !hasStructure {
		score -= 20
	}
	if wordCount < 150 {
		score -= 10
	}
	if !hasDialogue {
		score -= 5
	}
	if len(issues) > 3 {
		score -= 5
	}
	
	if score < 0 {
		score = 0
	}
	
	return QualityAssessment{
		HasProperEnding:   hasProperEnding,
		HasEndeMarker:     hasEndeMarker,
		IsComplete:        isComplete,
		HasClearStructure: hasStructure,
		HasDialogue:       hasDialogue,
		EndingIndicators:  endingIndicators,
		IssuesFound:       issues,
		QualityScore:      score,
	}
}

func printSummary(allResults []ModelResults) {
	log.Printf("\n%s\n", strings.Repeat("=", 80))
	log.Println("📊 ZUSAMMENFASSUNG DER TESTERGEBNISSE")
	log.Printf("%s\n\n", strings.Repeat("=", 80))

	// Header
	log.Printf("%-35s | %8s | %8s | %7s | %7s | %8s\n",
		"Modell", "Ø Zeit", "Ø Wörter", "GWS %", "Qual.", "ENDE ✓")
	log.Printf("%s\n", strings.Repeat("-", 80))

	for _, modelResult := range allResults {
		successfulTests := 0
		var totalTime, totalWords, totalGWSPerc, totalQualityScore float64
		endeMarkerCount := 0

		for _, test := range modelResult.Tests {
			if test.Success {
				successfulTests++
				totalTime += test.GenerationTime
				totalWords += float64(test.WordCount)
				totalGWSPerc += test.Grundwortschatz.Percentage
				totalQualityScore += test.Quality.QualityScore
				if test.Quality.HasEndeMarker {
					endeMarkerCount++
				}
			}
		}

		if successfulTests > 0 {
			avgTime := totalTime / float64(successfulTests)
			avgWords := totalWords / float64(successfulTests)
			avgGWS := totalGWSPerc / float64(successfulTests)
			avgQuality := totalQualityScore / float64(successfulTests)

			modelName := fmt.Sprintf("%s (%s)", modelResult.Model, modelResult.Provider)
			log.Printf("%-35s | %6.1fs | %8.0f | %6.1f%% | %6.0f | %3d/%d\n",
				modelName, avgTime, avgWords, avgGWS, avgQuality,
				endeMarkerCount, successfulTests)
		}
	}

	log.Printf("\n%s\n", strings.Repeat("=", 80))
	log.Printf("💾 Ergebnisse gespeichert in: test_results/\n")
	log.Printf("   - latest_results.json\n")
	log.Printf("   - latest_prompts.md\n")
	log.Printf("   - latest_full_report.md\n")
	log.Printf("%s\n\n", strings.Repeat("=", 80))
}

func saveResults(allResults []ModelResults) {
	// Create output directory
	if err := os.MkdirAll("test_results", 0755); err != nil {
		log.Printf("⚠️  Fehler beim Erstellen des Verzeichnisses: %v", err)
		return
	}

	filename := "test_results/latest_results.json"

	data, err := json.MarshalIndent(allResults, "", "  ")
	if err != nil {
		log.Printf("⚠️  Fehler beim JSON-Marshalling: %v", err)
		return
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		log.Printf("⚠️  Fehler beim Speichern: %v", err)
		return
	}

	log.Printf("💾 JSON gespeichert: %s", filename)
	
	// Save prompts separately for better readability
	savePrompts(allResults, "")
}

func savePrompts(allResults []ModelResults, timestamp string) {
	latestPromptFilename := "test_results/latest_prompts.md"
	
	var sb strings.Builder
	
	sb.WriteString("# 📝 Generierte Prompts - Modell-Vergleich\n\n")
	fmt.Fprintf(&sb, "**Datum:** %s\n\n", time.Now().Format("02.01.2006 15:04"))
	
	for _, modelResult := range allResults {
		fmt.Fprintf(&sb, "## Modell: %s (%s)\n\n", modelResult.Model, modelResult.Provider)
		
		for i, test := range modelResult.Tests {
			if !test.Success {
				continue
			}
			
			fmt.Fprintf(&sb, "### Test %d: %s\n\n", i+1, test.TestCase)
			
			sb.WriteString("#### System Prompt\n\n")
			sb.WriteString("```\n")
			sb.WriteString(test.SystemPrompt)
			sb.WriteString("\n```\n\n")
			
			sb.WriteString("#### User Prompt\n\n")
			sb.WriteString("```\n")
			sb.WriteString(test.UserPrompt)
			sb.WriteString("\n```\n\n")
			
			fmt.Fprintf(&sb, "**Ergebnis:** %d Wörter, %d Tokens, %.1fs\n\n", 
				test.WordCount, test.TokensUsed, test.GenerationTime)
			
			sb.WriteString("---\n\n")
		}
	}
	
	content := sb.String()
	
	// Save latest version only
	if err := os.WriteFile(latestPromptFilename, []byte(content), 0644); err != nil {
		log.Printf("⚠️  Fehler beim Speichern der Prompts: %v", err)
		return
	}
	log.Printf("📄 Prompts gespeichert: %s", latestPromptFilename)
}

func generateReport(allResults []ModelResults) {
	latestFilename := "test_results/latest_full_report.md"

	var sb strings.Builder

	// Header
	sb.WriteString("# 📊 Modell-Vergleichsbericht - Kindergeschichten\n\n")
	fmt.Fprintf(&sb, "**Datum:** %s\n", time.Now().Format("02.01.2006 15:04"))
	fmt.Fprintf(&sb, "**Getestete Modelle:** %d\n", len(allResults))
	fmt.Fprintf(&sb, "**Test-Cases:** %d\n\n", len(testCases))

	// Overview table
	sb.WriteString("## 📈 Gesamtübersicht\n\n")
	sb.WriteString("| Modell | Provider | Ø Zeit (s) | Ø Wörter | GWS % | Qualität | Erfolg |\n")
	sb.WriteString("|--------|----------|------------|----------|-------|----------|--------|\n")

	for _, modelResult := range allResults {
		successfulTests := 0
		var totalTime, totalWords, totalGWSPerc, totalQualityScore float64

		for _, test := range modelResult.Tests {
			if test.Success {
				successfulTests++
				totalTime += test.GenerationTime
				totalWords += float64(test.WordCount)
				totalGWSPerc += test.Grundwortschatz.Percentage
				totalQualityScore += test.Quality.QualityScore
			}
		}

		if successfulTests > 0 {
			avgTime := totalTime / float64(successfulTests)
			avgWords := totalWords / float64(successfulTests)
			avgGWS := totalGWSPerc / float64(successfulTests)
			avgQuality := totalQualityScore / float64(successfulTests)

			providerIcon := "🔧"
			switch modelResult.Provider {
			case "mistral-api":
				providerIcon = "🌐"
			case "ollama-cloud":
				providerIcon = "☁️"
			}

			fmt.Fprintf(&sb, "| %s | %s %s | %.1f | %.0f | %.1f%% | %.0f | %d/%d |\n",
				modelResult.Model, providerIcon, modelResult.Provider,
				avgTime, avgWords, avgGWS, avgQuality, successfulTests, len(modelResult.Tests))
		}
	}

	// Detailed results
	sb.WriteString("\n## 📝 Detaillierte Ergebnisse\n")

	for _, modelResult := range allResults {
		providerIcon := "🔧"
		switch modelResult.Provider {
		case "mistral-api":
			providerIcon = "🌐"
		case "ollama-cloud":
			providerIcon = "☁️"
		}

		fmt.Fprintf(&sb, "\n### %s %s - %s\n", providerIcon, modelResult.Provider, modelResult.Model)

		for _, test := range modelResult.Tests {
			fmt.Fprintf(&sb, "\n#### %s\n", test.TestCase)

			if test.Success {
				fmt.Fprintf(&sb, "- **Zeit:** %.1fs\n", test.GenerationTime)
				fmt.Fprintf(&sb, "- **Titel:** %s\n", test.Title)
				fmt.Fprintf(&sb, "- **Wörter:** %d\n", test.WordCount)
				fmt.Fprintf(&sb, "- **Absätze:** %d\n", test.ParagraphCount)
				fmt.Fprintf(&sb, "- **Dialoge:** %d\n", test.DialogueCount)
				fmt.Fprintf(&sb, "- **Grundwortschatz:** %d Wörter (%.1f%%)\n",
					test.Grundwortschatz.UniqueWords, test.Grundwortschatz.Percentage)
				fmt.Fprintf(&sb, "- **Tokens:** %d\n", test.TokensUsed)
				
				// Quality assessment
				fmt.Fprintf(&sb, "\n**Qualitätsbewertung:** %.0f/100\n", test.Quality.QualityScore)
				if test.Quality.IsComplete {
					sb.WriteString("- ✅ Geschichte ist vollständig\n")
				} else {
					sb.WriteString("- ❌ Geschichte ist unvollständig\n")
				}
				if test.Quality.HasEndeMarker {
					sb.WriteString("- ✅ ENDE-Marker vorhanden\n")
				} else {
					sb.WriteString("- ❌ ENDE-Marker fehlt\n")
				}
				if test.Quality.HasProperEnding {
					sb.WriteString("- ✅ Hat ein richtiges Ende\n")
				} else {
					sb.WriteString("- ❌ Ende fehlt oder ist abrupt\n")
				}
				if test.Quality.HasClearStructure {
					sb.WriteString("- ✅ Klare Struktur vorhanden\n")
				}
				if len(test.Quality.EndingIndicators) > 0 {
					fmt.Fprintf(&sb, "- 📝 Abschluss-Indikatoren: %s\n", strings.Join(test.Quality.EndingIndicators, ", "))
				}
				if len(test.Quality.IssuesFound) > 0 {
					fmt.Fprintf(&sb, "- ⚠️ Probleme: %s\n", strings.Join(test.Quality.IssuesFound, "; "))
				}
				
				fmt.Fprintf(&sb, "\n**Auszug:**\n> %s\n", test.StoryPreview)
			} else {
				fmt.Fprintf(&sb, "- **Fehler:** %s\n", test.Error)
			}
		}
	}

	// Recommendations
	sb.WriteString("\n## 🏆 Empfehlungen\n\n")

	if len(allResults) > 0 {
		// Find fastest model
		var fastest *ModelResults
		fastestAvg := 999999.0
		for i := range allResults {
			total := 0.0
			count := 0
			for _, test := range allResults[i].Tests {
				if test.Success {
					total += test.GenerationTime
					count++
				}
			}
			if count > 0 {
				avg := total / float64(count)
				if avg < fastestAvg {
					fastestAvg = avg
					fastest = &allResults[i]
				}
			}
		}
		if fastest != nil {
			fmt.Fprintf(&sb, "- **⚡ Schnellstes Modell:** %s (%s) - %.1fs\n",
				fastest.Model, fastest.Provider, fastestAvg)
		}

		// Find best GWS
		var bestGWS *ModelResults
		bestGWSAvg := 0.0
		for i := range allResults {
			total := 0.0
			count := 0
			for _, test := range allResults[i].Tests {
				if test.Success {
					total += test.Grundwortschatz.Percentage
					count++
				}
			}
			if count > 0 {
				avg := total / float64(count)
				if avg > bestGWSAvg {
					bestGWSAvg = avg
					bestGWS = &allResults[i]
				}
			}
		}
		if bestGWS != nil {
			fmt.Fprintf(&sb, "- **📚 Bester Grundwortschatz:** %s (%s) - %.1f%%\n",
				bestGWS.Model, bestGWS.Provider, bestGWSAvg)
		}
	}

	report := sb.String()

	// Save report (latest only)
	if err := os.WriteFile(latestFilename, []byte(report), 0644); err != nil {
		log.Printf("⚠️  Fehler beim Speichern des Reports: %v", err)
		return
	}
	log.Printf("📄 Report gespeichert: %s", latestFilename)
}

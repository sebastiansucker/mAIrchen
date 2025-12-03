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
	TokensUsed      int                     `json:"tokens_used,omitempty"`
	StoryPreview    string                  `json:"story_preview,omitempty"`
	Error           string                  `json:"error,omitempty"`
}

// GrundwortschatzAnalysis holds GWS analysis results
type GrundwortschatzAnalysis struct {
	UniqueWords      int      `json:"unique_words"`
	TotalOccurrences int      `json:"total_occurrences"`
	Percentage       float64  `json:"percentage"`
	TopWords         []string `json:"top_words"`
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
		Name:          "Klasse 1-2: Einfach - Tiere",
		Thema:         "Tiere und Natur",
		PersonenTiere: "Ein kleiner Hase",
		Ort:           "auf der Wiese",
		Stimmung:      "fröhlich",
		Laenge:        2,
		Klassenstufe:  "12",
	},
	{
		Name:          "Klasse 1-2: Mittel - Freundschaft",
		Thema:         "Freundschaft",
		PersonenTiere: "Ein Igel und ein Eichhörnchen",
		Ort:           "im Wald",
		Stimmung:      "herzlich",
		Laenge:        3,
		Klassenstufe:  "12",
	},
	{
		Name:          "Klasse 3-4: Einfach - Freundschaft",
		Thema:         "Freundschaft",
		PersonenTiere: "Ein kleiner Igel",
		Ort:           "im Wald",
		Stimmung:      "herzlich",
		Laenge:        3,
		Klassenstufe:  "34",
	},
	{
		Name:          "Klasse 3-4: Mittel - Abenteuer",
		Thema:         "Abenteuer",
		PersonenTiere: "Eine mutige Maus",
		Ort:           "in einer alten Mühle",
		Stimmung:      "spannend",
		Laenge:        5,
		Klassenstufe:  "34",
	},
	{
		Name:          "Klasse 3-4: Komplex - Zauber",
		Thema:         "Zauber und Magie",
		PersonenTiere: "Eine junge Hexe und ihr Kater",
		Ort:           "in einem verzauberten Garten",
		Stimmung:      "mysteriös",
		Laenge:        3,
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

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	startTime := time.Now()
	generatedStory, err := gen.Generate(ctx, req)
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
		Grundwortschatz: gwsAnalysis,
		TokensUsed:      generatedStory.TokensUsed,
		StoryPreview:    preview,
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

func saveResults(allResults []ModelResults) {
	// Create output directory
	if err := os.MkdirAll("test_results", 0755); err != nil {
		log.Printf("⚠️  Fehler beim Erstellen des Verzeichnisses: %v", err)
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("test_results/full_results_%s.json", timestamp)

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
}

func generateReport(allResults []ModelResults) {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("test_results/full_report_%s.md", timestamp)
	latestFilename := "test_results/latest_full_report.md"

	var sb strings.Builder

	// Header
	sb.WriteString("# 📊 Modell-Vergleichsbericht - Kindergeschichten\n\n")
	sb.WriteString(fmt.Sprintf("**Datum:** %s\n", time.Now().Format("02.01.2006 15:04")))
	sb.WriteString(fmt.Sprintf("**Getestete Modelle:** %d\n", len(allResults)))
	sb.WriteString(fmt.Sprintf("**Test-Cases:** %d\n\n", len(testCases)))

	// Overview table
	sb.WriteString("## 📈 Gesamtübersicht\n\n")
	sb.WriteString("| Modell | Provider | Ø Zeit (s) | Ø Wörter | GWS % | Erfolg |\n")
	sb.WriteString("|--------|----------|------------|----------|-------|--------|\n")

	for _, modelResult := range allResults {
		successfulTests := 0
		var totalTime, totalWords, totalGWSPerc float64

		for _, test := range modelResult.Tests {
			if test.Success {
				successfulTests++
				totalTime += test.GenerationTime
				totalWords += float64(test.WordCount)
				totalGWSPerc += test.Grundwortschatz.Percentage
			}
		}

		if successfulTests > 0 {
			avgTime := totalTime / float64(successfulTests)
			avgWords := totalWords / float64(successfulTests)
			avgGWS := totalGWSPerc / float64(successfulTests)

			providerIcon := "🔧"
			switch modelResult.Provider {
			case "mistral-api":
				providerIcon = "🌐"
			case "ollama-cloud":
				providerIcon = "☁️"
			}

			sb.WriteString(fmt.Sprintf("| %s | %s %s | %.1f | %.0f | %.1f%% | %d/%d |\n",
				modelResult.Model, providerIcon, modelResult.Provider,
				avgTime, avgWords, avgGWS, successfulTests, len(modelResult.Tests)))
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

		sb.WriteString(fmt.Sprintf("\n### %s %s - %s\n", providerIcon, modelResult.Provider, modelResult.Model))

		for _, test := range modelResult.Tests {
			sb.WriteString(fmt.Sprintf("\n#### %s\n", test.TestCase))

			if test.Success {
				sb.WriteString(fmt.Sprintf("- **Zeit:** %.1fs\n", test.GenerationTime))
				sb.WriteString(fmt.Sprintf("- **Titel:** %s\n", test.Title))
				sb.WriteString(fmt.Sprintf("- **Wörter:** %d\n", test.WordCount))
				sb.WriteString(fmt.Sprintf("- **Absätze:** %d\n", test.ParagraphCount))
				sb.WriteString(fmt.Sprintf("- **Dialoge:** %d\n", test.DialogueCount))
				sb.WriteString(fmt.Sprintf("- **Grundwortschatz:** %d Wörter (%.1f%%)\n",
					test.Grundwortschatz.UniqueWords, test.Grundwortschatz.Percentage))
				sb.WriteString(fmt.Sprintf("- **Tokens:** %d\n", test.TokensUsed))
				sb.WriteString(fmt.Sprintf("\n**Auszug:**\n> %s\n", test.StoryPreview))
			} else {
				sb.WriteString(fmt.Sprintf("- **Fehler:** %s\n", test.Error))
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
			sb.WriteString(fmt.Sprintf("- **⚡ Schnellstes Modell:** %s (%s) - %.1fs\n",
				fastest.Model, fastest.Provider, fastestAvg))
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
			sb.WriteString(fmt.Sprintf("- **📚 Bester Grundwortschatz:** %s (%s) - %.1f%%\n",
				bestGWS.Model, bestGWS.Provider, bestGWSAvg))
		}
	}

	report := sb.String()

	// Save report
	if err := os.WriteFile(filename, []byte(report), 0644); err != nil {
		log.Printf("⚠️  Fehler beim Speichern des Reports: %v", err)
		return
	}
	log.Printf("📄 Report gespeichert: %s", filename)

	// Save as latest
	if err := os.WriteFile(latestFilename, []byte(report), 0644); err != nil {
		log.Printf("⚠️  Fehler beim Speichern des Latest Reports: %v", err)
		return
	}
	log.Printf("📄 Latest Report: %s", latestFilename)
}

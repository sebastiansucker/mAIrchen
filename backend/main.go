package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sebastiansucker/mAIrchen/backend/pkg/config"
	"github.com/sebastiansucker/mAIrchen/backend/pkg/prompt"
	"github.com/sebastiansucker/mAIrchen/backend/pkg/story"
)

// Configuration
var (
	RateLimitPerIP   int
	RateLimitWindow  time.Duration
	GlobalDailyLimit int
	MaxStoryLength   int
	MaxDailyCost     float64
	CostPerRequest   = 0.0015
	AllowedOrigins   []string
	appConfig        *config.Config
	storyGenerator   *story.Generator
)

// Rate limiting storage
var (
	requestHistory     = make(map[string][]time.Time)
	globalRequestCount = struct {
		count     int
		resetTime time.Time
	}{count: 0, resetTime: time.Now().Add(24 * time.Hour)}
	dailyCost = struct {
		cost      float64
		resetTime time.Time
	}{cost: 0.0, resetTime: time.Now().Add(24 * time.Hour)}
	rateLimitLock sync.Mutex
)

// Models
type StoryResponse struct {
	Success         bool                   `json:"success"`
	Title           string                 `json:"title"`
	Story           string                 `json:"story"`
	Grundwortschatz []string               `json:"grundwortschatz"`
	Parameters      map[string]interface{} `json:"parameters"`
}

type RandomSuggestionsResponse struct {
	Thema         string `json:"thema"`
	PersonenTiere string `json:"personen_tiere"`
	Ort           string `json:"ort"`
	Stimmung      string `json:"stimmung"`
	Stil          string `json:"stil"`
}

type StatsResponse struct {
	GlobalRequestsToday int     `json:"global_requests_today"`
	GlobalLimit         int     `json:"global_limit"`
	EstimatedCostToday  float64 `json:"estimated_cost_today"`
	DailyBudget         float64 `json:"daily_budget"`
	BudgetRemaining     float64 `json:"budget_remaining"`
	RateLimitPerIP      int     `json:"rate_limit_per_ip"`
	ActiveIPs           int     `json:"active_ips"`
}

var suggestions = struct {
	Themen        []string
	PersonenTiere []string
	Orte          []string
	Stimmungen    []string
	Stile         []string
}{
	Themen: []string{
		"Freundschaft", "Abenteuer", "Zauber", "Tiere im Wald",
		"Eine Reise", "Ein Geheimnis", "Mut", "Hilfsbereitschaft", "Weihnachten", "Sommerferien", "Ein verlorener Schatz", "Magische Welten",
		"Die vier Jahreszeiten", "Ein besonderes Fest", "Die Kraft der Fantasie",
	},
	PersonenTiere: []string{
		"Ein kleiner Hase namens Erwin", "Eine mutige Prinzessin namens Helena", "Ein frecher Fuchs namens Felix",
		"Eine weise Eule", "Ein tapferere Ritterin names Hannelore", "Ein tapferer Ritter names Siegfried",
		"Ein neugieriges Eichhörnchen", "Ein kleines Mädchen namens Juna", "Ein junger Drache",
		"Eine zauberhafte Fee", "Der fröhliche Bär Klaus", "Ein kluger Junge", "Eine singende Nachtigall",
	},
	Orte: []string{
		"im Wald", "am See", "in einem Schloss", "auf einem Bauernhof",
		"in einem verzauberten Garten", "in den Bergen", "am Meer", "in einem Dorf", "im Zauberwald",
	},
	Stimmungen: []string{
		"fröhlich", "spannend", "mysteriös", "lustig",
		"abenteuerlich", "gemütlich", "aufregend", "herzlich",
	},
	Stile: []string{
		"Michael Ende", "Marc-Uwe Kling", "Astrid Lindgren", "Janosch",
		"Cornelia Funke", "Märchen", "Fabel", "Moderne Kindergeschichte",
	},
}

func init() {
	// Load configuration
	appConfig = config.LoadConfig()
	storyGenerator = story.NewGenerator(appConfig)

	// Load configuration from environment
	RateLimitPerIP = getEnvInt("RATE_LIMIT_PER_IP", 10)
	RateLimitWindow = time.Hour
	GlobalDailyLimit = getEnvInt("GLOBAL_DAILY_LIMIT", 1000)
	MaxStoryLength = getEnvInt("MAX_STORY_LENGTH", 15)
	MaxDailyCost = getEnvFloat("MAX_DAILY_COST", 5.0)

	originsStr := getEnv("ALLOWED_ORIGINS", "http://localhost,http://localhost:80,http://localhost:8080")
	AllowedOrigins = make([]string, 0)
	for _, o := range strings.Split(originsStr, ",") {
		AllowedOrigins = append(AllowedOrigins, strings.TrimSpace(o))
	}

	log.Println("mAIrchen Backend Go - Starting...")
	log.Printf("AI Provider: %s", appConfig.AIProvider)
	log.Printf("Model: %s", appConfig.DefaultModel)
	log.Printf("Base URL: %s", appConfig.OpenAIBaseURL)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

func getClientIP(c *gin.Context) string {
	forwarded := c.GetHeader("X-Forwarded-For")
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	return c.ClientIP()
}

func checkRateLimit(ip string) (bool, string) {
	rateLimitLock.Lock()
	defer rateLimitLock.Unlock()

	now := time.Now()

	// Reset global counter daily
	if now.After(globalRequestCount.resetTime) {
		globalRequestCount.count = 0
		globalRequestCount.resetTime = now.Add(24 * time.Hour)
	}

	// Reset daily cost
	if now.After(dailyCost.resetTime) {
		dailyCost.cost = 0.0
		dailyCost.resetTime = now.Add(24 * time.Hour)
	}

	// Check daily budget
	if dailyCost.cost >= MaxDailyCost {
		hoursUntilReset := int(time.Until(dailyCost.resetTime).Hours())
		return false, fmt.Sprintf("Tägliches Budget erreicht. Service pausiert für ~%dh.", hoursUntilReset)
	}

	// Check global limit
	if globalRequestCount.count >= GlobalDailyLimit {
		hoursUntilReset := int(time.Until(globalRequestCount.resetTime).Hours())
		return false, fmt.Sprintf("Tägliches Anfrage-Limit erreicht. Bitte in ~%dh erneut versuchen.", hoursUntilReset)
	}

	// Clean old requests
	cutoffTime := now.Add(-RateLimitWindow)
	var validRequests []time.Time
	for _, ts := range requestHistory[ip] {
		if ts.After(cutoffTime) {
			validRequests = append(validRequests, ts)
		}
	}
	requestHistory[ip] = validRequests

	// Check IP-specific limit
	if len(requestHistory[ip]) >= RateLimitPerIP {
		oldestExpires := requestHistory[ip][0].Add(RateLimitWindow)
		minutesUntilExpires := int(time.Until(oldestExpires).Minutes())
		return false, fmt.Sprintf("Zu viele Anfragen. Bitte warte ~%d Minuten.", minutesUntilExpires)
	}

	// Allow request
	requestHistory[ip] = append(requestHistory[ip], now)
	globalRequestCount.count++
	dailyCost.cost += CostPerRequest

	return true, ""
}

func main() {
	logLevel := getEnv("LOG_LEVEL", "INFO")
	if logLevel == "DEBUG" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// CORS middleware
	config := cors.DefaultConfig()
	config.AllowOrigins = AllowedOrigins
	config.AllowCredentials = true
	config.AllowMethods = []string{"GET", "POST"}
	config.AllowHeaders = []string{"Content-Type"}
	r.Use(cors.New(config))

	// Routes
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message":     "mAIrchen API - Märchen für Kinder",
			"ai_provider": appConfig.AIProvider,
			"model":       appConfig.DefaultModel,
		})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	r.GET("/api/random", handleRandomSuggestions)
	r.GET("/api/stats", handleStats)
	r.POST("/api/generate-story", handleGenerateStory)

	port := getEnv("PORT", "8000")
	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func handleRandomSuggestions(c *gin.Context) {
	c.JSON(http.StatusOK, RandomSuggestionsResponse{
		Thema:         suggestions.Themen[randomInt(len(suggestions.Themen))],
		PersonenTiere: suggestions.PersonenTiere[randomInt(len(suggestions.PersonenTiere))],
		Ort:           suggestions.Orte[randomInt(len(suggestions.Orte))],
		Stimmung:      suggestions.Stimmungen[randomInt(len(suggestions.Stimmungen))],
		Stil:          suggestions.Stile[randomInt(len(suggestions.Stile))],
	})
}

func handleStats(c *gin.Context) {
	rateLimitLock.Lock()
	defer rateLimitLock.Unlock()

	c.JSON(http.StatusOK, StatsResponse{
		GlobalRequestsToday: globalRequestCount.count,
		GlobalLimit:         GlobalDailyLimit,
		EstimatedCostToday:  roundFloat(dailyCost.cost, 2),
		DailyBudget:         MaxDailyCost,
		BudgetRemaining:     roundFloat(MaxDailyCost-dailyCost.cost, 2),
		RateLimitPerIP:      RateLimitPerIP,
		ActiveIPs:           len(requestHistory),
	})
}

func handleGenerateStory(c *gin.Context) {
	var req prompt.StoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}

	// Validate
	if req.Laenge > MaxStoryLength {
		c.JSON(http.StatusBadRequest, gin.H{"detail": fmt.Sprintf("Länge darf maximal %d Minuten sein", MaxStoryLength)})
		return
	}

	// Rate limiting
	clientIP := getClientIP(c)
	allowed, errMsg := checkRateLimit(clientIP)
	if !allowed {
		log.Printf("Rate Limit erreicht für IP %s: %s", clientIP, errMsg)
		c.JSON(http.StatusTooManyRequests, gin.H{"detail": errMsg})
		return
	}

	log.Printf("Story-Generierung gestartet - IP: %s", clientIP)

	// Generate story using the story generator
	ctx := c.Request.Context()
	generatedStory, err := storyGenerator.Generate(ctx, req)
	if err != nil {
		log.Printf("Fehler beim Generieren der Geschichte: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": fmt.Sprintf("Fehler beim Generieren der Geschichte: %v", err)})
		return
	}

	log.Println("API-Aufruf erfolgreich")
	log.Printf("Response Länge: %d Zeichen", len(generatedStory.Content))

	// Update cost tracking
	var actualCost float64
	switch appConfig.AIProvider {
	case "ollama-cloud":
		actualCost = float64(generatedStory.TokensUsed) / 1000 * 0.0005
	case "ollama-local":
		actualCost = 0.0
	default:
		actualCost = float64(generatedStory.TokensUsed) / 1000 * 0.001
	}

	rateLimitLock.Lock()
	dailyCost.cost += actualCost
	rateLimitLock.Unlock()

	c.JSON(http.StatusOK, StoryResponse{
		Success:         true,
		Title:           generatedStory.Title,
		Story:           generatedStory.Content,
		Grundwortschatz: generatedStory.Grundwortschatz,
		Parameters: map[string]interface{}{
			"thema":          req.Thema,
			"personen_tiere": req.PersonenTiere,
			"ort":            req.Ort,
			"stimmung":       req.Stimmung,
			"stil":           req.Stil,
			"laenge":         req.Laenge,
			"klassenstufe":   req.Klassenstufe,
		},
	})
}

func randomInt(max int) int {
	return rand.Intn(max)
}

func roundFloat(val float64, precision int) float64 {
	ratio := float64(1)
	for i := 0; i < precision; i++ {
		ratio *= 10
	}
	return float64(int(val*ratio+0.5)) / ratio
}

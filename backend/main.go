package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

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
	MaxFieldLength   = 200
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

// Streaming response events, written as newline-delimited JSON while the
// story is generated.
type streamTitleEvent struct {
	Type  string `json:"type"`
	Title string `json:"title"`
}

type streamChunkEvent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type streamDoneEvent struct {
	Type            string                 `json:"type"`
	Grundwortschatz []string               `json:"grundwortschatz"`
	TokensUsed      int                    `json:"tokens_used"`
	Parameters      map[string]interface{} `json:"parameters"`
}

type streamErrorEvent struct {
	Type   string `json:"type"`
	Detail string `json:"detail"`
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
		"Ein kleiner Hase namens Erwin, ein Hund namens Bruno", "Eine mutige Prinzessin namens Helena", "Ein frecher Fuchs namens Felix",
		"Eine weise Eule", "Ein tapferere Ritterin names Hannelore", "Ein tapferer Ritter names Siegfried",
		"Ein neugieriges Eichhörnchen", "Ein kleines Mädchen namens Juna", "Ein junger Drache",
		"Eine störrische Fee", "Der fröhliche Bär Klaus", "Ein kluger Junge", "Eine singende Nachtigall",
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
	// Nginx always overwrites X-Real-IP with the actual connecting address
	// ($remote_addr), so unlike X-Forwarded-For it cannot be spoofed by a
	// client to bypass per-IP rate limiting.
	if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
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

	// Clean old requests for this IP
	cutoffTime := now.Add(-RateLimitWindow)
	var validRequests []time.Time
	for _, ts := range requestHistory[ip] {
		if ts.After(cutoffTime) {
			validRequests = append(validRequests, ts)
		}
	}
	if len(validRequests) == 0 {
		delete(requestHistory, ip)
	} else {
		requestHistory[ip] = validRequests
	}

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

// cleanupStaleIPs removes IPs from requestHistory that have no requests left
// within the rate limit window, so IPs that never come back don't
// accumulate in memory forever.
func cleanupStaleIPs() {
	rateLimitLock.Lock()
	defer rateLimitLock.Unlock()

	cutoffTime := time.Now().Add(-RateLimitWindow)
	for ip, timestamps := range requestHistory {
		hasRecent := false
		for _, ts := range timestamps {
			if ts.After(cutoffTime) {
				hasRecent = true
				break
			}
		}
		if !hasRecent {
			delete(requestHistory, ip)
		}
	}
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

	go func() {
		ticker := time.NewTicker(RateLimitWindow)
		defer ticker.Stop()
		for range ticker.C {
			cleanupStaleIPs()
		}
	}()

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

// validateStoryRequest checks required fields, field lengths and story
// length against the documented limits. Returns an empty string if the
// request is valid, otherwise a user-facing error message.
func validateStoryRequest(req prompt.StoryRequest) string {
	if strings.TrimSpace(req.Thema) == "" ||
		strings.TrimSpace(req.PersonenTiere) == "" ||
		strings.TrimSpace(req.Ort) == "" ||
		strings.TrimSpace(req.Stimmung) == "" {
		return "Thema, Personen/Tiere, Ort und Stimmung sind Pflichtfelder"
	}

	fields := map[string]string{
		"thema":          req.Thema,
		"personen_tiere": req.PersonenTiere,
		"ort":            req.Ort,
		"stimmung":       req.Stimmung,
		"stil":           req.Stil,
	}
	for name, value := range fields {
		if utf8.RuneCountInString(value) > MaxFieldLength {
			return fmt.Sprintf("Feld '%s' darf maximal %d Zeichen lang sein", name, MaxFieldLength)
		}
	}

	if req.Laenge < 1 {
		return "Länge muss mindestens 1 Minute sein"
	}
	if req.Laenge > MaxStoryLength {
		return fmt.Sprintf("Länge darf maximal %d Minuten sein", MaxStoryLength)
	}

	return ""
}

func handleGenerateStory(c *gin.Context) {
	var req prompt.StoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}

	// Validate
	if errMsg := validateStoryRequest(req); errMsg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": errMsg})
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

	// Ab hier wird die Antwort als NDJSON gestreamt. Der HTTP-Status 200 wird
	// jetzt sofort committed und geflusht - ein späterer Fehler kann also
	// keinen neuen HTTP-Statuscode mehr senden (Header sind bereits raus).
	// Fehler nach diesem Punkt werden stattdessen als In-Band "error"-Event
	// übertragen; der Client muss dieses Event unabhängig vom (bereits
	// erfolgreichen) HTTP-Status als Fehlschlag behandeln.
	c.Writer.Header().Set("Content-Type", "application/x-ndjson")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	writeEvent := func(v interface{}) {
		b, err := json.Marshal(v)
		if err != nil {
			log.Printf("Fehler beim Marshalling des Stream-Events: %v", err)
			return
		}
		if _, err := c.Writer.Write(b); err != nil {
			log.Printf("Fehler beim Schreiben des Stream-Events: %v", err)
			return
		}
		if _, err := c.Writer.Write([]byte("\n")); err != nil {
			log.Printf("Fehler beim Schreiben des Stream-Events: %v", err)
			return
		}
		c.Writer.Flush()
	}

	// Generate story using the story generator
	ctx := c.Request.Context()
	generatedStory, err := storyGenerator.Generate(ctx, req, story.StreamCallbacks{
		OnTitle: func(title string) {
			writeEvent(streamTitleEvent{Type: "title", Title: title})
		},
		OnChunk: func(text string) {
			writeEvent(streamChunkEvent{Type: "chunk", Text: text})
		},
	})
	if err != nil {
		log.Printf("Fehler beim Generieren der Geschichte: %v", err)
		writeEvent(streamErrorEvent{Type: "error", Detail: fmt.Sprintf("Fehler beim Generieren der Geschichte: %v", err)})
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

	// checkRateLimit already reserved a flat CostPerRequest estimate when the
	// request was admitted (to guard the budget against bursts of concurrent
	// in-flight requests). Replace that reservation with the real cost now
	// that it's known, instead of adding on top of it.
	rateLimitLock.Lock()
	dailyCost.cost += actualCost - CostPerRequest
	rateLimitLock.Unlock()

	writeEvent(streamDoneEvent{
		Type:            "done",
		Grundwortschatz: generatedStory.Grundwortschatz,
		TokensUsed:      generatedStory.TokensUsed,
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

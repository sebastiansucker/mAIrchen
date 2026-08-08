package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sebastiansucker/mAIrchen/backend/pkg/config"
	"github.com/sebastiansucker/mAIrchen/backend/pkg/prompt"
	"github.com/sebastiansucker/mAIrchen/backend/pkg/story"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

// resetLimits puts the package-level rate limit state into a known, empty
// configuration and restores whatever init() left behind once the test ends.
// All rate limit state is global, so tests using it must not run in parallel.
func resetLimits(t *testing.T) {
	t.Helper()

	origPerIP, origWindow, origGlobal := RateLimitPerIP, RateLimitWindow, GlobalDailyLimit
	origMaxCost, origCostPerRequest, origMaxLen := MaxDailyCost, CostPerRequest, MaxStoryLength
	origHistory, origGlobalCount, origCost := requestHistory, globalRequestCount, dailyCost
	origConfig, origGenerator := appConfig, storyGenerator

	t.Cleanup(func() {
		rateLimitLock.Lock()
		defer rateLimitLock.Unlock()
		RateLimitPerIP, RateLimitWindow, GlobalDailyLimit = origPerIP, origWindow, origGlobal
		MaxDailyCost, CostPerRequest, MaxStoryLength = origMaxCost, origCostPerRequest, origMaxLen
		requestHistory, globalRequestCount, dailyCost = origHistory, origGlobalCount, origCost
		appConfig, storyGenerator = origConfig, origGenerator
	})

	rateLimitLock.Lock()
	defer rateLimitLock.Unlock()

	RateLimitPerIP = 3
	RateLimitWindow = time.Hour
	GlobalDailyLimit = 100
	MaxDailyCost = 5.0
	CostPerRequest = 0.0015
	MaxStoryLength = 15

	requestHistory = make(map[string][]time.Time)
	globalRequestCount.count = 0
	globalRequestCount.resetTime = time.Now().Add(24 * time.Hour)
	dailyCost.cost = 0.0
	dailyCost.resetTime = time.Now().Add(24 * time.Hour)
}

// ---------------------------------------------------------------------------
// getClientIP
// ---------------------------------------------------------------------------

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name     string
		realIP   string
		forwaded string
		remote   string
		expected string
	}{
		{
			name:     "X-Real-IP wins",
			realIP:   "203.0.113.5",
			remote:   "10.0.0.1:1234",
			expected: "203.0.113.5",
		},
		{
			name:     "X-Real-IP is trimmed",
			realIP:   "  203.0.113.5  ",
			remote:   "10.0.0.1:1234",
			expected: "203.0.113.5",
		},
		{
			// X-Forwarded-For is client-controlled; without an X-Real-IP from
			// nginx the peer address must be used, never the forwarded header.
			name:     "spoofed X-Forwarded-For is ignored without X-Real-IP",
			forwaded: "1.2.3.4",
			remote:   "10.0.0.1:1234",
			expected: "10.0.0.1",
		},
		{
			name:     "falls back to peer address",
			remote:   "192.0.2.9:5678",
			expected: "192.0.2.9",
		},
	}

	// Bind the context to the production engine: whether c.ClientIP() honours
	// X-Forwarded-For depends on the engine's trusted-proxy configuration.
	router := setupRouter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := gin.CreateTestContextOnly(httptest.NewRecorder(), router)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/generate-story", nil)
			c.Request.RemoteAddr = tt.remote
			if tt.realIP != "" {
				c.Request.Header.Set("X-Real-IP", tt.realIP)
			}
			if tt.forwaded != "" {
				c.Request.Header.Set("X-Forwarded-For", tt.forwaded)
			}

			if got := getClientIP(c); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// validateStoryRequest
// ---------------------------------------------------------------------------

func validRequest() prompt.StoryRequest {
	return prompt.StoryRequest{
		Thema:         "Freundschaft",
		PersonenTiere: "Ein Hase",
		Ort:           "im Wald",
		Stimmung:      "fröhlich",
		Laenge:        5,
		Klassenstufe:  "12",
	}
}

func TestValidateStoryRequest(t *testing.T) {
	resetLimits(t)

	tests := []struct {
		name        string
		mutate      func(*prompt.StoryRequest)
		expectError string
	}{
		{
			name:   "valid request",
			mutate: func(*prompt.StoryRequest) {},
		},
		{
			name:        "missing Thema",
			mutate:      func(r *prompt.StoryRequest) { r.Thema = "" },
			expectError: "Thema, Personen/Tiere, Ort und Stimmung sind Pflichtfelder",
		},
		{
			name:        "missing PersonenTiere",
			mutate:      func(r *prompt.StoryRequest) { r.PersonenTiere = "" },
			expectError: "Thema, Personen/Tiere, Ort und Stimmung sind Pflichtfelder",
		},
		{
			name:        "missing Ort",
			mutate:      func(r *prompt.StoryRequest) { r.Ort = "" },
			expectError: "Thema, Personen/Tiere, Ort und Stimmung sind Pflichtfelder",
		},
		{
			name:        "missing Stimmung",
			mutate:      func(r *prompt.StoryRequest) { r.Stimmung = "" },
			expectError: "Thema, Personen/Tiere, Ort und Stimmung sind Pflichtfelder",
		},
		{
			// Whitespace-only must not satisfy a required field.
			name:        "whitespace-only Thema",
			mutate:      func(r *prompt.StoryRequest) { r.Thema = "   \t  " },
			expectError: "Thema, Personen/Tiere, Ort und Stimmung sind Pflichtfelder",
		},
		{
			name:   "Stil is optional",
			mutate: func(r *prompt.StoryRequest) { r.Stil = "" },
		},
		{
			name:   "field at exactly MaxFieldLength is allowed",
			mutate: func(r *prompt.StoryRequest) { r.Thema = strings.Repeat("a", MaxFieldLength) },
		},
		{
			name:        "field one rune over the limit is rejected",
			mutate:      func(r *prompt.StoryRequest) { r.Thema = strings.Repeat("a", MaxFieldLength+1) },
			expectError: "Feld 'thema' darf maximal 200 Zeichen lang sein",
		},
		{
			// Umlauts are 2 bytes each: the limit must count runes, not bytes,
			// so 200 umlauts have to pass even though len() reports 400.
			name:   "200 multi-byte runes are allowed",
			mutate: func(r *prompt.StoryRequest) { r.Thema = strings.Repeat("ä", MaxFieldLength) },
		},
		{
			name:        "201 multi-byte runes are rejected",
			mutate:      func(r *prompt.StoryRequest) { r.Thema = strings.Repeat("ä", MaxFieldLength+1) },
			expectError: "Feld 'thema' darf maximal 200 Zeichen lang sein",
		},
		{
			name:        "over-long optional Stil is rejected",
			mutate:      func(r *prompt.StoryRequest) { r.Stil = strings.Repeat("a", MaxFieldLength+1) },
			expectError: "Feld 'stil' darf maximal 200 Zeichen lang sein",
		},
		{
			name:        "Laenge zero",
			mutate:      func(r *prompt.StoryRequest) { r.Laenge = 0 },
			expectError: "Länge muss mindestens 1 Minute sein",
		},
		{
			name:        "negative Laenge",
			mutate:      func(r *prompt.StoryRequest) { r.Laenge = -3 },
			expectError: "Länge muss mindestens 1 Minute sein",
		},
		{
			name:   "Laenge at the maximum is allowed",
			mutate: func(r *prompt.StoryRequest) { r.Laenge = MaxStoryLength },
		},
		{
			name:        "Laenge above the maximum",
			mutate:      func(r *prompt.StoryRequest) { r.Laenge = MaxStoryLength + 1 },
			expectError: "Länge darf maximal 15 Minuten sein",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validRequest()
			tt.mutate(&req)

			if got := validateStoryRequest(req); got != tt.expectError {
				t.Errorf("expected error %q, got %q", tt.expectError, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// checkRateLimit
// ---------------------------------------------------------------------------

func TestCheckRateLimit_AllowsUpToPerIPLimitThenBlocks(t *testing.T) {
	resetLimits(t)

	for i := 0; i < RateLimitPerIP; i++ {
		allowed, msg := checkRateLimit("10.0.0.1")
		if !allowed {
			t.Fatalf("request %d should have been allowed, got %q", i+1, msg)
		}
	}

	allowed, msg := checkRateLimit("10.0.0.1")
	if allowed {
		t.Fatal("request beyond the per-IP limit should have been blocked")
	}
	if !strings.Contains(msg, "Zu viele Anfragen") {
		t.Errorf("expected a per-IP rate limit message, got %q", msg)
	}
}

func TestCheckRateLimit_LimitIsPerIP(t *testing.T) {
	resetLimits(t)

	for i := 0; i < RateLimitPerIP; i++ {
		if allowed, _ := checkRateLimit("10.0.0.1"); !allowed {
			t.Fatalf("request %d for the first IP should have been allowed", i+1)
		}
	}
	if allowed, _ := checkRateLimit("10.0.0.1"); allowed {
		t.Fatal("first IP should be exhausted")
	}

	// A different IP must be unaffected by the first one's exhausted budget.
	if allowed, msg := checkRateLimit("10.0.0.2"); !allowed {
		t.Errorf("a second IP should still be allowed, got %q", msg)
	}
}

func TestCheckRateLimit_ForgetsRequestsOlderThanTheWindow(t *testing.T) {
	resetLimits(t)

	// Fill the IP's quota entirely with timestamps that already fell out of
	// the window; all of them must be discarded on the next check.
	stale := time.Now().Add(-RateLimitWindow - time.Minute)
	rateLimitLock.Lock()
	for i := 0; i < RateLimitPerIP; i++ {
		requestHistory["10.0.0.1"] = append(requestHistory["10.0.0.1"], stale)
	}
	rateLimitLock.Unlock()

	if allowed, msg := checkRateLimit("10.0.0.1"); !allowed {
		t.Fatalf("expired timestamps must not count towards the limit, got %q", msg)
	}

	rateLimitLock.Lock()
	remaining := len(requestHistory["10.0.0.1"])
	rateLimitLock.Unlock()
	if remaining != 1 {
		t.Errorf("expected only the new request to remain, got %d entries", remaining)
	}
}

func TestCheckRateLimit_GlobalDailyLimit(t *testing.T) {
	resetLimits(t)

	rateLimitLock.Lock()
	globalRequestCount.count = GlobalDailyLimit
	rateLimitLock.Unlock()

	// A fresh IP with no history at all must still be refused.
	allowed, msg := checkRateLimit("10.0.0.99")
	if allowed {
		t.Fatal("expected the global daily limit to block the request")
	}
	if !strings.Contains(msg, "Tägliches Anfrage-Limit erreicht") {
		t.Errorf("expected a global limit message, got %q", msg)
	}
}

func TestCheckRateLimit_DailyBudget(t *testing.T) {
	resetLimits(t)

	rateLimitLock.Lock()
	dailyCost.cost = MaxDailyCost
	rateLimitLock.Unlock()

	allowed, msg := checkRateLimit("10.0.0.99")
	if allowed {
		t.Fatal("expected the daily budget to block the request")
	}
	if !strings.Contains(msg, "Tägliches Budget erreicht") {
		t.Errorf("expected a budget message, got %q", msg)
	}
}

func TestCheckRateLimit_BudgetIsCheckedBeforeGlobalLimit(t *testing.T) {
	resetLimits(t)

	rateLimitLock.Lock()
	dailyCost.cost = MaxDailyCost
	globalRequestCount.count = GlobalDailyLimit
	rateLimitLock.Unlock()

	_, msg := checkRateLimit("10.0.0.99")
	if !strings.Contains(msg, "Tägliches Budget erreicht") {
		t.Errorf("budget exhaustion should be reported first, got %q", msg)
	}
}

func TestCheckRateLimit_ResetsCountersAfterResetTime(t *testing.T) {
	resetLimits(t)

	rateLimitLock.Lock()
	globalRequestCount.count = GlobalDailyLimit
	globalRequestCount.resetTime = time.Now().Add(-time.Minute)
	dailyCost.cost = MaxDailyCost
	dailyCost.resetTime = time.Now().Add(-time.Minute)
	rateLimitLock.Unlock()

	if allowed, msg := checkRateLimit("10.0.0.1"); !allowed {
		t.Fatalf("expected the request to be allowed after the daily reset, got %q", msg)
	}

	rateLimitLock.Lock()
	defer rateLimitLock.Unlock()
	if globalRequestCount.count != 1 {
		t.Errorf("expected the global counter to restart at 1, got %d", globalRequestCount.count)
	}
	if globalRequestCount.resetTime.Before(time.Now()) {
		t.Error("expected the global reset time to be moved into the future")
	}
	if dailyCost.resetTime.Before(time.Now()) {
		t.Error("expected the cost reset time to be moved into the future")
	}
}

func TestCheckRateLimit_ReservesCostPerRequest(t *testing.T) {
	resetLimits(t)

	// Each admitted request reserves a flat estimate up front so that a burst
	// of concurrent in-flight requests cannot overshoot the budget.
	for i := 1; i <= 3; i++ {
		if allowed, _ := checkRateLimit("10.0.0.1"); !allowed {
			t.Fatalf("request %d should have been allowed", i)
		}

		rateLimitLock.Lock()
		got := dailyCost.cost
		rateLimitLock.Unlock()

		want := CostPerRequest * float64(i)
		if diff := got - want; diff > 1e-9 || diff < -1e-9 {
			t.Errorf("after %d requests expected reserved cost %f, got %f", i, want, got)
		}
	}
}

func TestCheckRateLimit_BlockedRequestsDoNotConsumeBudget(t *testing.T) {
	resetLimits(t)

	for i := 0; i < RateLimitPerIP; i++ {
		checkRateLimit("10.0.0.1")
	}

	rateLimitLock.Lock()
	costBefore := dailyCost.cost
	countBefore := globalRequestCount.count
	rateLimitLock.Unlock()

	if allowed, _ := checkRateLimit("10.0.0.1"); allowed {
		t.Fatal("expected the request to be blocked")
	}

	rateLimitLock.Lock()
	defer rateLimitLock.Unlock()
	if dailyCost.cost != costBefore {
		t.Errorf("a blocked request must not reserve budget: %f -> %f", costBefore, dailyCost.cost)
	}
	if globalRequestCount.count != countBefore {
		t.Errorf("a blocked request must not count towards the global limit: %d -> %d", countBefore, globalRequestCount.count)
	}
}

// ---------------------------------------------------------------------------
// cleanupStaleIPs
// ---------------------------------------------------------------------------

func TestCleanupStaleIPs(t *testing.T) {
	resetLimits(t)

	now := time.Now()
	rateLimitLock.Lock()
	requestHistory["stale"] = []time.Time{now.Add(-RateLimitWindow - time.Minute)}
	requestHistory["fresh"] = []time.Time{now.Add(-time.Minute)}
	requestHistory["mixed"] = []time.Time{now.Add(-RateLimitWindow - time.Minute), now.Add(-time.Minute)}
	requestHistory["empty"] = nil
	rateLimitLock.Unlock()

	cleanupStaleIPs()

	rateLimitLock.Lock()
	defer rateLimitLock.Unlock()

	if _, ok := requestHistory["stale"]; ok {
		t.Error("an IP with only expired timestamps should have been removed")
	}
	if _, ok := requestHistory["empty"]; ok {
		t.Error("an IP with no timestamps should have been removed")
	}
	if _, ok := requestHistory["fresh"]; !ok {
		t.Error("an IP with a recent timestamp must be kept")
	}
	if _, ok := requestHistory["mixed"]; !ok {
		t.Error("an IP with at least one recent timestamp must be kept")
	}
}

// ---------------------------------------------------------------------------
// roundFloat
// ---------------------------------------------------------------------------

func TestRoundFloat(t *testing.T) {
	tests := []struct {
		name      string
		val       float64
		precision int
		expected  float64
	}{
		{name: "rounds down", val: 1.234, precision: 2, expected: 1.23},
		{name: "rounds up", val: 1.236, precision: 2, expected: 1.24},
		{name: "rounds half away from zero", val: 1.235, precision: 2, expected: 1.24},
		{name: "zero precision", val: 1.6, precision: 0, expected: 2},
		{name: "already exact", val: 5.0, precision: 2, expected: 5},
		{name: "zero", val: 0, precision: 2, expected: 0},
		// BudgetRemaining is MaxDailyCost - dailyCost.cost and goes negative
		// once the real token cost overshoots the reserved estimate, so the
		// negative branch is reachable in production.
		{name: "negative rounds down", val: -1.238, precision: 2, expected: -1.24},
		{name: "negative rounds up", val: -1.232, precision: 2, expected: -1.23},
		{name: "negative half away from zero", val: -1.235, precision: 2, expected: -1.24},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := roundFloat(tt.val, tt.precision)
			if diff := got - tt.expected; diff > 1e-9 || diff < -1e-9 {
				t.Errorf("roundFloat(%v, %d) = %v, want %v", tt.val, tt.precision, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// newTestRouter uses the production router so the tests cover the real
// middleware chain and proxy configuration, not a hand-rolled subset.
func newTestRouter() *gin.Engine {
	return setupRouter()
}

func TestSetupRouter_RegistersRoutes(t *testing.T) {
	want := map[string]string{
		"GET /":                    "",
		"GET /health":              "",
		"GET /api/random":          "",
		"GET /api/stats":           "",
		"POST /api/generate-story": "",
	}

	for _, route := range setupRouter().Routes() {
		delete(want, route.Method+" "+route.Path)
	}

	for missing := range want {
		t.Errorf("route %s is not registered", missing)
	}
}

func TestHealthEndpoint(t *testing.T) {
	w := httptest.NewRecorder()
	newTestRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"healthy"`) {
		t.Errorf("expected a healthy status, got %q", w.Body.String())
	}
}

func TestHandleRandomSuggestions(t *testing.T) {
	w := httptest.NewRecorder()
	newTestRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/random", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var got RandomSuggestionsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	// Every field must come from its own suggestion list - a mix-up between
	// the lists would still produce a well-formed response.
	fields := []struct {
		name string
		got  string
		pool []string
	}{
		{"thema", got.Thema, suggestions.Themen},
		{"personen_tiere", got.PersonenTiere, suggestions.PersonenTiere},
		{"ort", got.Ort, suggestions.Orte},
		{"stimmung", got.Stimmung, suggestions.Stimmungen},
		{"stil", got.Stil, suggestions.Stile},
	}
	for _, f := range fields {
		found := false
		for _, candidate := range f.pool {
			if candidate == f.got {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s = %q is not from its suggestion list", f.name, f.got)
		}
	}
}

func TestHandleStats(t *testing.T) {
	resetLimits(t)

	rateLimitLock.Lock()
	globalRequestCount.count = 7
	dailyCost.cost = 1.234
	requestHistory["10.0.0.1"] = []time.Time{time.Now()}
	requestHistory["10.0.0.2"] = []time.Time{time.Now()}
	rateLimitLock.Unlock()

	w := httptest.NewRecorder()
	newTestRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/stats", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var got StatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if got.GlobalRequestsToday != 7 {
		t.Errorf("expected 7 requests today, got %d", got.GlobalRequestsToday)
	}
	if got.GlobalLimit != GlobalDailyLimit {
		t.Errorf("expected global limit %d, got %d", GlobalDailyLimit, got.GlobalLimit)
	}
	if got.EstimatedCostToday != 1.23 {
		t.Errorf("expected cost rounded to 1.23, got %v", got.EstimatedCostToday)
	}
	if got.DailyBudget != MaxDailyCost {
		t.Errorf("expected budget %v, got %v", MaxDailyCost, got.DailyBudget)
	}
	if got.BudgetRemaining != 3.77 {
		t.Errorf("expected remaining budget 3.77, got %v", got.BudgetRemaining)
	}
	if got.RateLimitPerIP != RateLimitPerIP {
		t.Errorf("expected per-IP limit %d, got %d", RateLimitPerIP, got.RateLimitPerIP)
	}
	if got.ActiveIPs != 2 {
		t.Errorf("expected 2 active IPs, got %d", got.ActiveIPs)
	}
}

func postStory(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/generate-story", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", "10.0.0.1")

	w := httptest.NewRecorder()
	newTestRouter().ServeHTTP(w, req)
	return w
}

func TestHandleGenerateStory_MalformedJSON(t *testing.T) {
	resetLimits(t)

	w := postStory(t, `{"thema": `)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	rateLimitLock.Lock()
	defer rateLimitLock.Unlock()
	if dailyCost.cost != 0 {
		t.Errorf("a rejected request must not reserve budget, got %f", dailyCost.cost)
	}
}

func TestHandleGenerateStory_ValidationError(t *testing.T) {
	resetLimits(t)

	w := postStory(t, `{"thema":"","personen_tiere":"Hase","ort":"Wald","stimmung":"froh","laenge":5}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if !strings.Contains(body["detail"], "Pflichtfelder") {
		t.Errorf("expected a validation message, got %q", body["detail"])
	}

	// Validation runs before rate limiting, so an invalid request must not
	// consume any of the caller's quota.
	rateLimitLock.Lock()
	defer rateLimitLock.Unlock()
	if len(requestHistory) != 0 {
		t.Errorf("an invalid request must not count towards the rate limit, got %v", requestHistory)
	}
}

func TestHandleGenerateStory_RateLimited(t *testing.T) {
	resetLimits(t)

	rateLimitLock.Lock()
	dailyCost.cost = MaxDailyCost
	rateLimitLock.Unlock()

	w := postStory(t, `{"thema":"Mut","personen_tiere":"Hase","ort":"Wald","stimmung":"froh","laenge":5,"klassenstufe":"12"}`)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if !strings.Contains(body["detail"], "Budget") {
		t.Errorf("expected a budget message, got %q", body["detail"])
	}
}

// fakeLLM serves an OpenAI-compatible SSE stream so the handler can be driven
// end to end without a real provider.
func fakeLLM(t *testing.T, content string, totalTokens int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)

		for _, fragment := range strings.SplitAfter(content, "\n") {
			if fragment == "" {
				continue
			}
			chunk, _ := json.Marshal(map[string]any{
				"choices": []map[string]any{{"index": 0, "delta": map[string]string{"content": fragment}}},
			})
			_, _ = fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}

		usage, _ := json.Marshal(map[string]any{
			"choices": []map[string]any{},
			"usage":   map[string]int{"total_tokens": totalTokens},
		})
		_, _ = fmt.Fprintf(w, "data: %s\n\n", usage)
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	t.Cleanup(server.Close)
	return server
}

// readNDJSON decodes the newline-delimited events the handler streams back.
func readNDJSON(t *testing.T, body string) []map[string]any {
	t.Helper()
	var events []map[string]any
	scanner := bufio.NewScanner(strings.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("stream line is not valid JSON: %q (%v)", line, err)
		}
		events = append(events, event)
	}
	return events
}

func TestHandleGenerateStory_StreamsTitleChunksAndDone(t *testing.T) {
	resetLimits(t)

	server := fakeLLM(t, "TITEL: Der kleine Hase\nEs war einmal der kleine Hase.\nENDE\n", 2000)

	rateLimitLock.Lock()
	appConfig = &config.Config{AIProvider: "openai", DefaultModel: "test-model", OpenAIBaseURL: server.URL}
	storyGenerator = story.NewGenerator(appConfig)
	rateLimitLock.Unlock()

	w := postStory(t, `{"thema":"Mut","personen_tiere":"Hase","ort":"Wald","stimmung":"froh","laenge":5,"klassenstufe":"12"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/x-ndjson" {
		t.Errorf("expected NDJSON content type, got %q", ct)
	}

	events := readNDJSON(t, w.Body.String())
	if len(events) < 2 {
		t.Fatalf("expected at least a title and a done event, got %d: %v", len(events), events)
	}

	if events[0]["type"] != "title" {
		t.Errorf("expected the first event to be the title, got %v", events[0])
	}
	if events[0]["title"] != "Der kleine Hase" {
		t.Errorf("expected the parsed title, got %v", events[0]["title"])
	}

	done := events[len(events)-1]
	if done["type"] != "done" {
		t.Fatalf("expected the last event to be 'done', got %v", done)
	}
	if got := done["tokens_used"]; got != float64(2000) {
		t.Errorf("expected 2000 tokens reported, got %v", got)
	}
	if words, ok := done["grundwortschatz"].([]any); !ok || len(words) == 0 {
		t.Errorf("expected Grundwortschatz matches in the done event, got %v", done["grundwortschatz"])
	}

	// The done event echoes the request so the frontend can label the story.
	params, ok := done["parameters"].(map[string]any)
	if !ok {
		t.Fatalf("expected parameters in the done event, got %v", done["parameters"])
	}
	if params["thema"] != "Mut" || params["klassenstufe"] != "12" || params["laenge"] != float64(5) {
		t.Errorf("done event does not echo the request parameters: %v", params)
	}

	// No error event may appear on the success path - the HTTP status is
	// already committed at 200, so errors are only visible in-band.
	for _, e := range events {
		if e["type"] == "error" {
			t.Errorf("unexpected error event on the success path: %v", e)
		}
	}
}

func TestHandleGenerateStory_ReplacesCostReservationWithActualCost(t *testing.T) {
	tests := []struct {
		name       string
		provider   string
		tokens     int
		expectCost float64
	}{
		// The reservation of CostPerRequest must be replaced by the real cost,
		// not added to it: 2000 tokens at 0.001 EUR/1000 = 0.002 EUR.
		{name: "default provider", provider: "openai", tokens: 2000, expectCost: 0.002},
		{name: "ollama-cloud is cheaper", provider: "ollama-cloud", tokens: 2000, expectCost: 0.001},
		{name: "ollama-local is free", provider: "ollama-local", tokens: 2000, expectCost: 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetLimits(t)

			server := fakeLLM(t, "TITEL: Titel\nEine kleine Geschichte mit der Katze.\nENDE\n", tt.tokens)

			rateLimitLock.Lock()
			appConfig = &config.Config{AIProvider: tt.provider, DefaultModel: "test-model", OpenAIBaseURL: server.URL}
			storyGenerator = story.NewGenerator(appConfig)
			rateLimitLock.Unlock()

			w := postStory(t, `{"thema":"Mut","personen_tiere":"Katze","ort":"Wald","stimmung":"froh","laenge":5,"klassenstufe":"12"}`)
			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", w.Code)
			}

			rateLimitLock.Lock()
			got := dailyCost.cost
			rateLimitLock.Unlock()

			if diff := got - tt.expectCost; diff > 1e-9 || diff < -1e-9 {
				t.Errorf("expected daily cost %f after reconciliation, got %f", tt.expectCost, got)
			}
		})
	}
}

func TestHandleGenerateStory_ReportsProviderFailureAsInBandErrorEvent(t *testing.T) {
	resetLimits(t)

	// The provider is unreachable, so the stream never opens. The handler has
	// not written its headers yet at that point, but it must still answer with
	// an "error" event rather than a non-200 status.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	rateLimitLock.Lock()
	appConfig = &config.Config{AIProvider: "openai", DefaultModel: "test-model", OpenAIBaseURL: server.URL}
	storyGenerator = story.NewGenerator(appConfig)
	rateLimitLock.Unlock()

	w := postStory(t, `{"thema":"Mut","personen_tiere":"Hase","ort":"Wald","stimmung":"froh","laenge":5,"klassenstufe":"12"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected the already-committed 200 status, got %d", w.Code)
	}

	events := readNDJSON(t, w.Body.String())
	if len(events) != 1 {
		t.Fatalf("expected exactly one error event, got %d: %v", len(events), events)
	}
	if events[0]["type"] != "error" {
		t.Errorf("expected an error event, got %v", events[0])
	}
	if detail, _ := events[0]["detail"].(string); !strings.Contains(detail, "Fehler beim Generieren") {
		t.Errorf("expected a generation error detail, got %q", detail)
	}

	// checkRateLimit reserved CostPerRequest when the request was admitted.
	// Since generation failed, nothing was actually spent, so that
	// reservation must be refunded rather than left standing - otherwise a
	// misconfigured provider or an upstream outage inflates dailyCost.cost on
	// every failed attempt until the daily budget trips and the service
	// pauses itself despite having spent nothing.
	rateLimitLock.Lock()
	got := dailyCost.cost
	rateLimitLock.Unlock()
	if got != 0 {
		t.Errorf("a failed generation must refund the reserved cost, got dailyCost.cost = %f", got)
	}
}

// ---------------------------------------------------------------------------
// Env helpers
// ---------------------------------------------------------------------------

func TestGetEnvHelpers(t *testing.T) {
	t.Run("getEnv returns the value when set", func(t *testing.T) {
		t.Setenv("MAIRCHEN_TEST_STR", "value")
		if got := getEnv("MAIRCHEN_TEST_STR", "fallback"); got != "value" {
			t.Errorf("expected 'value', got %q", got)
		}
	})

	t.Run("getEnv falls back when empty", func(t *testing.T) {
		t.Setenv("MAIRCHEN_TEST_STR", "")
		if got := getEnv("MAIRCHEN_TEST_STR", "fallback"); got != "fallback" {
			t.Errorf("expected 'fallback', got %q", got)
		}
	})

	t.Run("getEnvInt parses the value", func(t *testing.T) {
		t.Setenv("MAIRCHEN_TEST_INT", "42")
		if got := getEnvInt("MAIRCHEN_TEST_INT", 7); got != 42 {
			t.Errorf("expected 42, got %d", got)
		}
	})

	t.Run("getEnvInt falls back on garbage", func(t *testing.T) {
		t.Setenv("MAIRCHEN_TEST_INT", "not-a-number")
		if got := getEnvInt("MAIRCHEN_TEST_INT", 7); got != 7 {
			t.Errorf("expected the default 7 for an unparsable value, got %d", got)
		}
	})

	t.Run("getEnvFloat parses the value", func(t *testing.T) {
		t.Setenv("MAIRCHEN_TEST_FLOAT", "2.5")
		if got := getEnvFloat("MAIRCHEN_TEST_FLOAT", 1.0); got != 2.5 {
			t.Errorf("expected 2.5, got %v", got)
		}
	})

	t.Run("getEnvFloat falls back on garbage", func(t *testing.T) {
		t.Setenv("MAIRCHEN_TEST_FLOAT", "not-a-float")
		if got := getEnvFloat("MAIRCHEN_TEST_FLOAT", 1.0); got != 1.0 {
			t.Errorf("expected the default 1.0 for an unparsable value, got %v", got)
		}
	})
}

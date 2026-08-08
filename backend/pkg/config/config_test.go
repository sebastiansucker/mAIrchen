package config

import (
	"os"
	"testing"
)

func TestLoadConfig_OllamaCloud(t *testing.T) {
	// Setup
	_ = os.Setenv("AI_PROVIDER", "ollama-cloud")
	_ = os.Setenv("OLLAMA_API_KEY", "test-key-123")
	_ = os.Setenv("OLLAMA_MODEL", "test-model")
	defer func() {
		_ = os.Unsetenv("AI_PROVIDER")
		_ = os.Unsetenv("OLLAMA_API_KEY")
		_ = os.Unsetenv("OLLAMA_MODEL")
	}()

	// Execute
	cfg := LoadConfig()

	// Assert
	if cfg.AIProvider != "ollama-cloud" {
		t.Errorf("Expected AIProvider 'ollama-cloud', got '%s'", cfg.AIProvider)
	}
	if cfg.OpenAIAPIKey != "test-key-123" {
		t.Errorf("Expected OpenAIAPIKey 'test-key-123', got '%s'", cfg.OpenAIAPIKey)
	}
	if cfg.OpenAIBaseURL != "https://ollama.com/v1" {
		t.Errorf("Expected OpenAIBaseURL 'https://ollama.com/v1', got '%s'", cfg.OpenAIBaseURL)
	}
	if cfg.DefaultModel != "test-model" {
		t.Errorf("Expected DefaultModel 'test-model', got '%s'", cfg.DefaultModel)
	}
}

func TestLoadConfig_OllamaLocal(t *testing.T) {
	// Setup
	_ = os.Setenv("AI_PROVIDER", "ollama-local")
	_ = os.Setenv("OLLAMA_BASE_URL", "http://localhost:11434/v1")
	_ = os.Setenv("OLLAMA_MODEL", "llama2")
	defer func() {
		_ = os.Unsetenv("AI_PROVIDER")
		_ = os.Unsetenv("OLLAMA_BASE_URL")
		_ = os.Unsetenv("OLLAMA_MODEL")
	}()

	// Execute
	cfg := LoadConfig()

	// Assert
	if cfg.AIProvider != "ollama-local" {
		t.Errorf("Expected AIProvider 'ollama-local', got '%s'", cfg.AIProvider)
	}
	if cfg.OpenAIAPIKey != "dummy-key" {
		t.Errorf("Expected OpenAIAPIKey 'dummy-key', got '%s'", cfg.OpenAIAPIKey)
	}
	if cfg.OpenAIBaseURL != "http://localhost:11434/v1" {
		t.Errorf("Expected OpenAIBaseURL 'http://localhost:11434/v1', got '%s'", cfg.OpenAIBaseURL)
	}
	if cfg.DefaultModel != "llama2" {
		t.Errorf("Expected DefaultModel 'llama2', got '%s'", cfg.DefaultModel)
	}
}

func TestLoadConfig_OpenAI(t *testing.T) {
	// Setup
	_ = os.Setenv("AI_PROVIDER", "openai")
	_ = os.Setenv("OPENAI_API_KEY", "sk-test-key")
	_ = os.Setenv("OPENAI_MODEL", "gpt-4-turbo")
	defer func() {
		_ = os.Unsetenv("AI_PROVIDER")
		_ = os.Unsetenv("OPENAI_API_KEY")
		_ = os.Unsetenv("OPENAI_MODEL")
	}()

	// Execute
	cfg := LoadConfig()

	// Assert
	if cfg.AIProvider != "openai" {
		t.Errorf("Expected AIProvider 'openai', got '%s'", cfg.AIProvider)
	}
	if cfg.OpenAIAPIKey != "sk-test-key" {
		t.Errorf("Expected OpenAIAPIKey 'sk-test-key', got '%s'", cfg.OpenAIAPIKey)
	}
	if cfg.OpenAIBaseURL != "https://api.openai.com/v1" {
		t.Errorf("Expected OpenAIBaseURL 'https://api.openai.com/v1', got '%s'", cfg.OpenAIBaseURL)
	}
	if cfg.DefaultModel != "gpt-4-turbo" {
		t.Errorf("Expected DefaultModel 'gpt-4-turbo', got '%s'", cfg.DefaultModel)
	}
}

func TestLoadConfig_OpenAI_CustomBaseURL(t *testing.T) {
	// Setup
	_ = os.Setenv("AI_PROVIDER", "openai")
	_ = os.Setenv("OPENAI_API_KEY", "mistral-key")
	_ = os.Setenv("OPENAI_BASE_URL", "https://api.mistral.ai/v1")
	_ = os.Setenv("OPENAI_MODEL", "mistral-small-latest")
	defer func() {
		_ = os.Unsetenv("AI_PROVIDER")
		_ = os.Unsetenv("OPENAI_API_KEY")
		_ = os.Unsetenv("OPENAI_BASE_URL")
		_ = os.Unsetenv("OPENAI_MODEL")
	}()

	// Execute
	cfg := LoadConfig()

	// Assert - AI_PROVIDER=openai must still respect an explicit OPENAI_BASE_URL
	if cfg.OpenAIBaseURL != "https://api.mistral.ai/v1" {
		t.Errorf("Expected OpenAIBaseURL 'https://api.mistral.ai/v1', got '%s'", cfg.OpenAIBaseURL)
	}
}

// An unset AI_PROVIDER defaults to "openai" and therefore takes the "openai"
// switch branch - not the default branch below. Pointing Mistral at the
// OpenAI-compatible client is done via OPENAI_BASE_URL, which that branch
// honours.
func TestLoadConfig_UnsetProviderDefaultsToOpenAIBranch(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "mistral-key")
	t.Setenv("OPENAI_BASE_URL", "https://api.mistral.ai/v1")
	t.Setenv("OPENAI_MODEL", "mistral-large")
	_ = os.Unsetenv("AI_PROVIDER")

	cfg := LoadConfig()

	if cfg.AIProvider != "openai" {
		t.Errorf("Expected AIProvider 'openai' (default), got '%s'", cfg.AIProvider)
	}
	if cfg.OpenAIAPIKey != "mistral-key" {
		t.Errorf("Expected OpenAIAPIKey 'mistral-key', got '%s'", cfg.OpenAIAPIKey)
	}
	if cfg.OpenAIBaseURL != "https://api.mistral.ai/v1" {
		t.Errorf("Expected OpenAIBaseURL 'https://api.mistral.ai/v1', got '%s'", cfg.OpenAIBaseURL)
	}
	if cfg.DefaultModel != "mistral-large" {
		t.Errorf("Expected DefaultModel 'mistral-large', got '%s'", cfg.DefaultModel)
	}
}

// The switch's default branch is only reachable with an AI_PROVIDER value that
// is not one of the three documented ones; it falls back to Mistral.
func TestLoadConfig_UnknownProviderFallsBackToMistral(t *testing.T) {
	t.Setenv("AI_PROVIDER", "some-unknown-provider")
	t.Setenv("OPENAI_API_KEY", "fallback-key")
	_ = os.Unsetenv("OPENAI_BASE_URL")
	_ = os.Unsetenv("OPENAI_MODEL")

	cfg := LoadConfig()

	if cfg.AIProvider != "some-unknown-provider" {
		t.Errorf("Expected the raw AIProvider to be preserved, got '%s'", cfg.AIProvider)
	}
	if cfg.OpenAIAPIKey != "fallback-key" {
		t.Errorf("Expected OpenAIAPIKey 'fallback-key', got '%s'", cfg.OpenAIAPIKey)
	}
	if cfg.OpenAIBaseURL != "https://api.mistral.ai/v1" {
		t.Errorf("Expected the Mistral base URL, got '%s'", cfg.OpenAIBaseURL)
	}
	if cfg.DefaultModel != "mistral-large-latest" {
		t.Errorf("Expected DefaultModel 'mistral-large-latest', got '%s'", cfg.DefaultModel)
	}
}

// Each documented provider has to supply working defaults without any
// provider-specific environment variables set.
func TestLoadConfig_DefaultsWithoutOptionalEnv(t *testing.T) {
	tests := []struct {
		provider    string
		expectURL   string
		expectModel string
		expectKey   string
	}{
		{provider: "ollama-cloud", expectURL: "https://ollama.com/v1", expectModel: "ministral-3:8b-cloud", expectKey: "dummy-key"},
		{provider: "ollama-local", expectURL: "http://localhost:11434/v1", expectModel: "mistral:7b", expectKey: "dummy-key"},
		{provider: "openai", expectURL: "https://api.openai.com/v1", expectModel: "gpt-4", expectKey: ""},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			t.Setenv("AI_PROVIDER", tt.provider)
			for _, key := range []string{"OLLAMA_API_KEY", "OLLAMA_BASE_URL", "OLLAMA_MODEL", "OPENAI_API_KEY", "OPENAI_BASE_URL", "OPENAI_MODEL"} {
				_ = os.Unsetenv(key)
			}

			cfg := LoadConfig()

			if cfg.OpenAIBaseURL != tt.expectURL {
				t.Errorf("Expected base URL '%s', got '%s'", tt.expectURL, cfg.OpenAIBaseURL)
			}
			if cfg.DefaultModel != tt.expectModel {
				t.Errorf("Expected model '%s', got '%s'", tt.expectModel, cfg.DefaultModel)
			}
			if cfg.OpenAIAPIKey != tt.expectKey {
				t.Errorf("Expected API key '%s', got '%s'", tt.expectKey, cfg.OpenAIAPIKey)
			}
		})
	}
}

func TestGetEnv_WithValue(t *testing.T) {
	// Setup
	_ = os.Setenv("TEST_KEY", "test-value")
	defer func() { _ = os.Unsetenv("TEST_KEY") }()

	// Execute
	result := getEnv("TEST_KEY", "default-value")

	// Assert
	if result != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", result)
	}
}

func TestGetEnv_WithDefault(t *testing.T) {
	// Setup - ensure key doesn't exist
	_ = os.Unsetenv("NON_EXISTENT_KEY")

	// Execute
	result := getEnv("NON_EXISTENT_KEY", "default-value")

	// Assert
	if result != "default-value" {
		t.Errorf("Expected 'default-value', got '%s'", result)
	}
}

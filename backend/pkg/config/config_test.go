package config

import (
	"os"
	"testing"
)

func TestLoadConfig_OllamaCloud(t *testing.T) {
	// Setup
	os.Setenv("AI_PROVIDER", "ollama-cloud")
	os.Setenv("OLLAMA_API_KEY", "test-key-123")
	os.Setenv("OLLAMA_MODEL", "test-model")
	defer func() {
		os.Unsetenv("AI_PROVIDER")
		os.Unsetenv("OLLAMA_API_KEY")
		os.Unsetenv("OLLAMA_MODEL")
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
	os.Setenv("AI_PROVIDER", "ollama-local")
	os.Setenv("OLLAMA_BASE_URL", "http://localhost:11434/v1")
	os.Setenv("OLLAMA_MODEL", "llama2")
	defer func() {
		os.Unsetenv("AI_PROVIDER")
		os.Unsetenv("OLLAMA_BASE_URL")
		os.Unsetenv("OLLAMA_MODEL")
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
	os.Setenv("AI_PROVIDER", "openai")
	os.Setenv("OPENAI_API_KEY", "sk-test-key")
	os.Setenv("OPENAI_MODEL", "gpt-4-turbo")
	defer func() {
		os.Unsetenv("AI_PROVIDER")
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("OPENAI_MODEL")
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

func TestLoadConfig_DefaultMistral(t *testing.T) {
	// Setup - no AI_PROVIDER set
	os.Unsetenv("AI_PROVIDER")
	os.Setenv("OPENAI_API_KEY", "mistral-key")
	os.Setenv("OPENAI_BASE_URL", "https://api.mistral.ai/v1")
	os.Setenv("OPENAI_MODEL", "mistral-large")
	defer func() {
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("OPENAI_BASE_URL")
		os.Unsetenv("OPENAI_MODEL")
	}()

	// Execute
	cfg := LoadConfig()

	// Assert - should default to "openai" provider but use mistral settings
	if cfg.AIProvider != "openai" {
		t.Errorf("Expected AIProvider 'openai' (default), got '%s'", cfg.AIProvider)
	}
	if cfg.OpenAIAPIKey != "mistral-key" {
		t.Errorf("Expected OpenAIAPIKey 'mistral-key', got '%s'", cfg.OpenAIAPIKey)
	}
}

func TestGetEnv_WithValue(t *testing.T) {
	// Setup
	os.Setenv("TEST_KEY", "test-value")
	defer os.Unsetenv("TEST_KEY")

	// Execute
	result := getEnv("TEST_KEY", "default-value")

	// Assert
	if result != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", result)
	}
}

func TestGetEnv_WithDefault(t *testing.T) {
	// Setup - ensure key doesn't exist
	os.Unsetenv("NON_EXISTENT_KEY")

	// Execute
	result := getEnv("NON_EXISTENT_KEY", "default-value")

	// Assert
	if result != "default-value" {
		t.Errorf("Expected 'default-value', got '%s'", result)
	}
}

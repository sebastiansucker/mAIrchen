package config

import (
	"os"
)

// Config holds all application configuration
type Config struct {
	AIProvider    string
	OpenAIAPIKey  string
	OpenAIBaseURL string
	DefaultModel  string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	aiProvider := getEnv("AI_PROVIDER", "openai")
	
	cfg := &Config{
		AIProvider: aiProvider,
	}
	
	// Provider-specific configuration
	switch aiProvider {
	case "ollama-cloud":
		cfg.OpenAIAPIKey = getEnv("OLLAMA_API_KEY", "dummy-key")
		cfg.OpenAIBaseURL = "https://ollama.com/v1"
		cfg.DefaultModel = getEnv("OLLAMA_MODEL", "ministral-3:8b-cloud")
	case "ollama-local":
		cfg.OpenAIAPIKey = "dummy-key"
		cfg.OpenAIBaseURL = getEnv("OLLAMA_BASE_URL", "http://localhost:11434/v1")
		cfg.DefaultModel = getEnv("OLLAMA_MODEL", "mistral:7b")
	case "openai":
		cfg.OpenAIAPIKey = getEnv("OPENAI_API_KEY", "")
		cfg.OpenAIBaseURL = "https://api.openai.com/v1"
		cfg.DefaultModel = getEnv("OPENAI_MODEL", "gpt-4")
	default:
		cfg.OpenAIAPIKey = getEnv("OPENAI_API_KEY", "")
		cfg.OpenAIBaseURL = getEnv("OPENAI_BASE_URL", "https://api.mistral.ai/v1")
		cfg.DefaultModel = getEnv("OPENAI_MODEL", "mistral-large-latest")
	}
	
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

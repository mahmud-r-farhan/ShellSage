package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds application configuration
type Config struct {
	APIKey  string
	Model   string
	BaseURL string
}

// LoadConfig loads configuration from environment variables and .env file
func LoadConfig() (*Config, error) {
	// Try loading from .env file (ignore error if not found)
	_ = godotenv.Load()

	// Get API key from environment
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY not found in environment or .env file")
	}

	// Get model from environment or use default
	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = "openrouter/free" // Better default model
	}

	// Get base URL from environment or use default
	baseURL := os.Getenv("OPENROUTER_BASE_URL")
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}

	return &Config{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: baseURL,
	}, nil
}

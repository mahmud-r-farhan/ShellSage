package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// ProviderType identifies the LLM backend
type ProviderType string

const (
	ProviderOpenRouter ProviderType = "openrouter"
	ProviderOpenAI     ProviderType = "openai"
	ProviderAnthropic  ProviderType = "anthropic"
	ProviderGemini     ProviderType = "gemini"
	ProviderGroq       ProviderType = "groq"
	ProviderDeepSeek   ProviderType = "deepseek"
	ProviderOllama     ProviderType = "ollama"
	ProviderCustom     ProviderType = "custom"
)

// ProviderConfig holds configuration for a specific provider
type ProviderConfig struct {
	APIKey  string `json:"api_key,omitempty"`
	BaseURL string `json:"base_url,omitempty"`
	Model   string `json:"model,omitempty"`
}

// Config holds global and provider-specific configurations
type Config struct {
	ActiveProvider  ProviderType                    `json:"active_provider"`
	Providers       map[ProviderType]ProviderConfig `json:"providers"`
	DefaultTemp     float64                         `json:"default_temp"`
	StreamResponses bool                            `json:"stream_responses"`
	AutoSaveHistory bool                            `json:"auto_save_history"`
	ConfigPath      string                          `json:"-"`
}

// DefaultModels mapped to each provider
var DefaultModels = map[ProviderType]string{
	ProviderOpenRouter: "openrouter/free",
	ProviderOpenAI:     "gpt-4o-mini",
	ProviderAnthropic:  "claude-3-5-haiku-latest",
	ProviderGemini:     "gemini-2.0-flash",
	ProviderGroq:       "llama-3.3-70b-versatile",
	ProviderDeepSeek:   "deepseek-chat",
	ProviderOllama:     "llama3.2:latest",
	ProviderCustom:     "default-model",
}

// DefaultBaseURLs mapped to each provider
var DefaultBaseURLs = map[ProviderType]string{
	ProviderOpenRouter: "https://openrouter.ai/api/v1",
	ProviderOpenAI:     "https://api.openai.com/v1",
	ProviderAnthropic:  "https://api.anthropic.com/v1",
	ProviderGemini:     "https://generativelanguage.googleapis.com/v1beta/openai",
	ProviderGroq:       "https://api.groq.com/openai/v1",
	ProviderDeepSeek:   "https://api.deepseek.com/v1",
	ProviderOllama:     "http://localhost:11434/v1",
	ProviderCustom:     "http://localhost:8080/v1",
}

// GetUserConfigDir returns the directory for storing user configuration
func GetUserConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".shellsage")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// DefaultConfig returns a sane default configuration
func DefaultConfig() *Config {
	providers := make(map[ProviderType]ProviderConfig)
	for p, model := range DefaultModels {
		providers[p] = ProviderConfig{
			BaseURL: DefaultBaseURLs[p],
			Model:   model,
		}
	}

	return &Config{
		ActiveProvider:  ProviderOpenRouter,
		Providers:       providers,
		DefaultTemp:     0.7,
		StreamResponses: true,
		AutoSaveHistory: true,
	}
}

// LoadConfig loads configuration from ~/.shellsage/config.json, falling back to .env / env vars
func LoadConfig() (*Config, error) {
	// Try loading .env first (ignore if missing)
	_ = godotenv.Load()

	cfg := DefaultConfig()
	configDir := GetUserConfigDir()
	configFilePath := filepath.Join(configDir, "config.json")
	cfg.ConfigPath = configFilePath

	// Load JSON config file if present
	if data, err := os.ReadFile(configFilePath); err == nil {
		_ = json.Unmarshal(data, cfg)
	}

	// Environment variable overrides (Backwards compatibility with v1/v2 & 12-factor apps)
	applyEnvOverrides(cfg)

	// Ensure active provider has defaults if unset
	if cfg.ActiveProvider == "" {
		cfg.ActiveProvider = ProviderOpenRouter
	}

	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	// Check OpenRouter
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		p := cfg.Providers[ProviderOpenRouter]
		p.APIKey = key
		if m := os.Getenv("OPENROUTER_MODEL"); m != "" {
			p.Model = m
		}
		if u := os.Getenv("OPENROUTER_BASE_URL"); u != "" {
			p.BaseURL = u
		}
		cfg.Providers[ProviderOpenRouter] = p
	}

	// Check OpenAI
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		p := cfg.Providers[ProviderOpenAI]
		p.APIKey = key
		if m := os.Getenv("OPENAI_MODEL"); m != "" {
			p.Model = m
		}
		if u := os.Getenv("OPENAI_BASE_URL"); u != "" {
			p.BaseURL = u
		}
		cfg.Providers[ProviderOpenAI] = p
	}

	// Check Anthropic
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		p := cfg.Providers[ProviderAnthropic]
		p.APIKey = key
		if m := os.Getenv("ANTHROPIC_MODEL"); m != "" {
			p.Model = m
		}
		cfg.Providers[ProviderAnthropic] = p
	}

	// Check Gemini
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		p := cfg.Providers[ProviderGemini]
		p.APIKey = key
		if m := os.Getenv("GEMINI_MODEL"); m != "" {
			p.Model = m
		}
		cfg.Providers[ProviderGemini] = p
	}

	// Check Groq
	if key := os.Getenv("GROQ_API_KEY"); key != "" {
		p := cfg.Providers[ProviderGroq]
		p.APIKey = key
		if m := os.Getenv("GROQ_MODEL"); m != "" {
			p.Model = m
		}
		cfg.Providers[ProviderGroq] = p
	}

	// Check DeepSeek
	if key := os.Getenv("DEEPSEEK_API_KEY"); key != "" {
		p := cfg.Providers[ProviderDeepSeek]
		p.APIKey = key
		if m := os.Getenv("DEEPSEEK_MODEL"); m != "" {
			p.Model = m
		}
		cfg.Providers[ProviderDeepSeek] = p
	}

	// Check Ollama
	if u := os.Getenv("OLLAMA_BASE_URL"); u != "" {
		p := cfg.Providers[ProviderOllama]
		p.BaseURL = u
		if m := os.Getenv("OLLAMA_MODEL"); m != "" {
			p.Model = m
		}
		cfg.Providers[ProviderOllama] = p
	}

	// Active Provider override
	if act := os.Getenv("SHELLSAGE_PROVIDER"); act != "" {
		cfg.ActiveProvider = ProviderType(strings.ToLower(act))
	}
}

// Save saves current configuration to ~/.shellsage/config.json
func (c *Config) Save() error {
	path := c.ConfigPath
	if path == "" {
		path = filepath.Join(GetUserConfigDir(), "config.json")
		c.ConfigPath = path
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", path, err)
	}

	return nil
}

// GetActiveProviderConfig returns the ProviderConfig for the currently active provider
func (c *Config) GetActiveProviderConfig() (ProviderConfig, error) {
	pConfig, ok := c.Providers[c.ActiveProvider]
	if !ok {
		return ProviderConfig{}, fmt.Errorf("provider %s not found in configuration", c.ActiveProvider)
	}
	return pConfig, nil
}

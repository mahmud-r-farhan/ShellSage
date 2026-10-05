package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"shellsage/internal/provider"
)

// ProviderType identifies the LLM backend (registry-driven; see internal/provider).
type ProviderType string

const (
	ProviderOpenRouter ProviderType = "openrouter"
	ProviderOpenAI     ProviderType = "openai"
	ProviderAnthropic  ProviderType = "anthropic"
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

	// v4 additions for real-world usage -------------------------------
	// MaxContextTokens bounds the chat context sent to the model.
	// Oldest turns are trimmed automatically when exceeded (0 disables).
	MaxContextTokens int `json:"max_context_tokens,omitempty"`
	// ApproveMode controls agent autonomy: "ask" (default, gate risky
	// tools interactively), "yolo" (auto-approve) or "read-only".
	ApproveMode string `json:"approve_mode,omitempty"`
	// AgentMaxSteps caps ReAct iterations for agent runs (0 = default 12).
	AgentMaxSteps int `json:"agent_max_steps,omitempty"`
	// RequestTimeoutSec bounds individual HTTP requests to the LLM API.
	RequestTimeoutSec int `json:"request_timeout_sec,omitempty"`
	// MaxRetries overrides transient-failure retries (default 3).
	MaxRetries *int `json:"max_retries,omitempty"`
	// --------------------------------------------------------------

	ConfigPath string `json:"-"`
}

// DefaultMaxContextTokens is the safe default budget for chat context.
const DefaultMaxContextTokens = 60_000

// DefaultModels is derived from the provider registry (kept for compatibility).
var DefaultModels = map[ProviderType]string{}

// DefaultBaseURLs is derived from the provider registry (kept for compatibility).
var DefaultBaseURLs = map[ProviderType]string{}

func init() {
	for _, d := range provider.All() {
		DefaultModels[ProviderType(d.ID)] = d.DefaultModel
		DefaultBaseURLs[ProviderType(d.ID)] = d.DefaultBaseURL
	}
}

// GetUserConfigDir returns the directory for storing user configuration.
// Honors SHELLSAGE_HOME for sandboxes/containers.
func GetUserConfigDir() string {
	if dir := os.Getenv("SHELLSAGE_HOME"); dir != "" {
		_ = os.MkdirAll(dir, 0700)
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".shellsage")
	_ = os.MkdirAll(dir, 0700)
	return dir
}

// DefaultConfig returns a sane default configuration (registry-driven).
func DefaultConfig() *Config {
	providers := make(map[ProviderType]ProviderConfig)
	for _, d := range provider.All() {
		providers[ProviderType(d.ID)] = ProviderConfig{
			BaseURL: d.DefaultBaseURL,
			Model:   d.DefaultModel,
		}
	}

	return &Config{
		ActiveProvider:   ProviderOpenRouter,
		Providers:        providers,
		DefaultTemp:      0.7,
		StreamResponses:  true,
		AutoSaveHistory:  true,
		MaxContextTokens: DefaultMaxContextTokens,
		ApproveMode:      "ask",
	}
}

// LoadConfig loads configuration from ~/.shellsage/config.json, falling back
// to .env / env vars. Path resolution: explicit override > $SHELLSAGE_CONFIG >
// ~/.shellsage/config.json.
func LoadConfig() (*Config, error) {
	return LoadConfigAt("")
}

// LoadConfigAt is LoadConfig with an explicit config-file override.
func LoadConfigAt(explicitPath string) (*Config, error) {
	// Try loading .env first (ignore if missing)
	_ = godotenv.Load()

	cfg := DefaultConfig()

	configFilePath := explicitPath
	if configFilePath == "" {
		if envPath := os.Getenv("SHELLSAGE_CONFIG"); envPath != "" {
			configFilePath = envPath
		} else {
			configFilePath = filepath.Join(GetUserConfigDir(), "config.json")
		}
	}
	cfg.ConfigPath = configFilePath

	// Load JSON config file if present
	if data, err := os.ReadFile(configFilePath); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			// Corrupt file: fall back to defaults but keep the path for Save().
			fresh := DefaultConfig()
			fresh.ConfigPath = configFilePath
			cfg = fresh
		}
	}

	// Environment variable overrides (12-factor & backwards compatibility)
	applyEnvOverrides(cfg)

	// Ensure active provider is valid; fall back to a registry default.
	if _, ok := cfg.Providers[cfg.ActiveProvider]; !ok {
		if id, valid := provider.NormalizeID(string(cfg.ActiveProvider)); valid {
			cfg.ActiveProvider = ProviderType(id)
		} else if cfg.ActiveProvider != "" {
			// Unknown id in config file: keep it (provider.New will report it),
			// but ensure a map entry exists so menus don't panic.
			if cfg.Providers == nil {
				cfg.Providers = map[ProviderType]ProviderConfig{}
			}
			cfg.Providers[cfg.ActiveProvider] = ProviderConfig{}
		} else {
			cfg.ActiveProvider = ProviderOpenRouter
		}
	}

	// Ensure every registry provider has an entry (config written before an
	// upgrade may be missing newly added providers).
	for _, d := range provider.All() {
		if _, ok := cfg.Providers[ProviderType(d.ID)]; !ok {
			cfg.Providers[ProviderType(d.ID)] = ProviderConfig{
				BaseURL: d.DefaultBaseURL,
				Model:   d.DefaultModel,
			}
		}
	}

	if cfg.MaxContextTokens == 0 {
		cfg.MaxContextTokens = DefaultMaxContextTokens
	}
	if cfg.ApproveMode == "" {
		cfg.ApproveMode = "ask"
	}

	return cfg, nil
}

// applyEnvOverrides walks the provider registry and applies
// <PREFIX>_API_KEY / <PREFIX>_MODEL / <PREFIX>_BASE_URL for every provider,
// plus a few documented aliases (GITHUB_TOKEN, AZURE_OPENAI_ENDPOINT, ...).
func applyEnvOverrides(cfg *Config) {
	for _, d := range provider.All() {
		id := ProviderType(d.ID)
		p := cfg.Providers[id]

		keyNames := append([]string{d.EnvPrefix + "_API_KEY"}, d.KeyEnvs...)
		if key := firstEnv(keyNames...); key != "" {
			p.APIKey = key
		}
		if m := os.Getenv(d.EnvPrefix + "_MODEL"); m != "" {
			p.Model = m
		}
		if u := os.Getenv(d.EnvPrefix + "_BASE_URL"); u != "" {
			p.BaseURL = u
		} else if d.BaseURLEnv != "" {
			if u := os.Getenv(d.BaseURLEnv); u != "" {
				p.BaseURL = u
			}
		}

		cfg.Providers[id] = p
	}

	// Global overrides
	if act := os.Getenv("SHELLSAGE_PROVIDER"); act != "" {
		if id, ok := provider.NormalizeID(act); ok {
			cfg.ActiveProvider = ProviderType(id)
		} else {
			cfg.ActiveProvider = ProviderType(strings.ToLower(act))
		}
	}
	if m := os.Getenv("SHELLSAGE_MODEL"); m != "" {
		p := cfg.Providers[cfg.ActiveProvider]
		p.Model = m
		cfg.Providers[cfg.ActiveProvider] = p
	}
	if t := os.Getenv("SHELLSAGE_TEMP"); t != "" {
		if v, err := strconv.ParseFloat(t, 64); err == nil && v >= 0 && v <= 2 {
			cfg.DefaultTemp = v
		}
	}
	if s := os.Getenv("SHELLSAGE_STREAM"); s != "" {
		if v, err := strconv.ParseBool(s); err == nil {
			cfg.StreamResponses = v
		}
	}
	if a := os.Getenv("SHELLSAGE_AUTO_SAVE"); a != "" {
		if v, err := strconv.ParseBool(a); err == nil {
			cfg.AutoSaveHistory = v
		}
	}
	if mc := os.Getenv("SHELLSAGE_MAX_CONTEXT_TOKENS"); mc != "" {
		if v, err := strconv.Atoi(mc); err == nil && v >= 0 {
			cfg.MaxContextTokens = v
		}
	}
	if am := os.Getenv("SHELLSAGE_APPROVE"); am != "" {
		cfg.ApproveMode = strings.ToLower(am)
	}
}

func firstEnv(names ...string) string {
	for _, n := range names {
		if n == "" {
			continue
		}
		if v := os.Getenv(n); v != "" {
			return v
		}
	}
	return ""
}

// Save persists the configuration to ConfigPath with owner-only permissions
// (the file contains API keys).
func (c *Config) Save() error {
	path := c.ConfigPath
	if path == "" {
		path = filepath.Join(GetUserConfigDir(), "config.json")
		c.ConfigPath = path
	}

	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create config dir %s: %w", dir, err)
		}
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", path, err)
	}
	// Tighten perms even if the file pre-existed with looser modes.
	_ = os.Chmod(path, 0600)

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

// ActiveSelection resolves the active provider into an explicit
// provider.Selection (filling registry defaults for empty fields).
func (c *Config) ActiveSelection() (provider.Selection, error) {
	pCfg, err := c.GetActiveProviderConfig()
	if err != nil {
		return provider.Selection{}, err
	}
	if pCfg.BaseURL == "" || pCfg.Model == "" {
		if d, ok := provider.ByID(string(c.ActiveProvider)); ok {
			if pCfg.BaseURL == "" {
				pCfg.BaseURL = d.DefaultBaseURL
			}
			if pCfg.Model == "" {
				pCfg.Model = d.DefaultModel
			}
		}
	}
	return provider.Selection{
		ID:      string(c.ActiveProvider),
		APIKey:  pCfg.APIKey,
		BaseURL: pCfg.BaseURL,
		Model:   pCfg.Model,
	}, nil
}

// NewActiveProvider instantiates the provider implementation for the active config.
func (c *Config) NewActiveProvider() (provider.Provider, error) {
	sel, err := c.ActiveSelection()
	if err != nil {
		return nil, err
	}
	p, err := provider.New(sel)
	if err != nil {
		return nil, err
	}
	// Propagate transport tuning.
	if c.RequestTimeoutSec > 0 {
		if t, ok := p.(interface{ SetTimeout(time.Duration) }); ok {
			t.SetTimeout(time.Duration(c.RequestTimeoutSec) * time.Second)
		}
	}
	if c.MaxRetries != nil {
		if r, ok := p.(interface{ SetMaxRetries(int) }); ok {
			r.SetMaxRetries(*c.MaxRetries)
		}
	}
	return p, nil
}

// SetProviderKey updates the API key for one provider in memory (and saves).
func (c *Config) SetProviderKey(id, key string) error {
	if _, ok := provider.ByID(id); !ok {
		return fmt.Errorf("unknown provider %q", id)
	}
	p := c.Providers[ProviderType(id)]
	p.APIKey = key
	c.Providers[ProviderType(id)] = p
	return nil
}

// SetProviderModel updates the default model for one provider.
func (c *Config) SetProviderModel(id, model string) error {
	if _, ok := provider.ByID(id); !ok {
		return fmt.Errorf("unknown provider %q", id)
	}
	resolved, _ := provider.NormalizeID(id)
	p := c.Providers[ProviderType(resolved)]
	p.Model = model
	c.Providers[ProviderType(resolved)] = p
	return nil
}

// SetProviderBaseURL updates the endpoint for one provider.
func (c *Config) SetProviderBaseURL(id, url string) error {
	if _, ok := provider.ByID(id); !ok {
		return fmt.Errorf("unknown provider %q", id)
	}
	resolved, _ := provider.NormalizeID(id)
	p := c.Providers[ProviderType(resolved)]
	p.BaseURL = strings.TrimSuffix(url, "/")
	c.Providers[ProviderType(resolved)] = p
	return nil
}

// MaskKey renders an API key safely for display.
func MaskKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	if len(key) <= 10 {
		return "***"
	}
	return key[:4] + "…" + key[len(key)-4:]
}

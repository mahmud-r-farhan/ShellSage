package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ActiveProvider != ProviderOpenRouter {
		t.Errorf("expected default active provider openrouter, got %s", cfg.ActiveProvider)
	}
	if len(cfg.Providers) == 0 {
		t.Errorf("expected default providers list to not be empty")
	}
	if cfg.DefaultTemp != 0.7 {
		t.Errorf("expected default temp 0.7, got %f", cfg.DefaultTemp)
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	cfg := DefaultConfig()
	cfg.ConfigPath = configPath
	cfg.ActiveProvider = ProviderOpenAI
	p := cfg.Providers[ProviderOpenAI]
	p.APIKey = "sk-test-key"
	p.Model = "gpt-4o"
	cfg.Providers[ProviderOpenAI] = p

	err := cfg.Save()
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("config file was not created")
	}

	// Read and verify
	pCfg, err := cfg.GetActiveProviderConfig()
	if err != nil {
		t.Fatalf("failed to get active provider config: %v", err)
	}
	if pCfg.APIKey != "sk-test-key" {
		t.Errorf("expected API key sk-test-key, got %s", pCfg.APIKey)
	}
	if pCfg.Model != "gpt-4o" {
		t.Errorf("expected model gpt-4o, got %s", pCfg.Model)
	}
}

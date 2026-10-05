package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGenericEnvOverridesForNewProviders(t *testing.T) {
	t.Setenv("MISTRAL_API_KEY", "mk-test")
	t.Setenv("MISTRAL_MODEL", "codestral-latest")
	t.Setenv("CEREBRAS_API_KEY", "cb-test")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := cfg.Providers["mistral"].APIKey; got != "mk-test" {
		t.Errorf("mistral key override failed: %q", got)
	}
	if got := cfg.Providers["mistral"].Model; got != "codestral-latest" {
		t.Errorf("mistral model override failed: %q", got)
	}
	if got := cfg.Providers["cerebras"].APIKey; got != "cb-test" {
		t.Errorf("cerebras key override failed: %q", got)
	}
}

func TestKeyAliasGITHUBToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_alias")
	cfg, _ := LoadConfig()
	if got := cfg.Providers["github"].APIKey; got != "ghp_alias" {
		t.Errorf("GITHUB_TOKEN alias not applied: %q", got)
	}
}

func TestAzureEndpointEnv(t *testing.T) {
	t.Setenv("AZURE_OPENAI_ENDPOINT", "https://corp.openai.azure.com")
	t.Setenv("AZURE_OPENAI_API_KEY", "az-key")
	cfg, _ := LoadConfig()
	p := cfg.Providers["azure"]
	if p.BaseURL != "https://corp.openai.azure.com" {
		t.Errorf("azure base url: %q", p.BaseURL)
	}
	if p.APIKey != "az-key" {
		t.Errorf("azure key: %q", p.APIKey)
	}
}

func TestProviderAliasNormalization(t *testing.T) {
	t.Setenv("SHELLSAGE_PROVIDER", "Claude")
	cfg, _ := LoadConfig()
	if cfg.ActiveProvider != "anthropic" {
		t.Errorf("alias 'Claude' should normalize to anthropic, got %s", cfg.ActiveProvider)
	}
}

func TestSaveUsesOwnerOnlyPermissions(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.ConfigPath = filepath.Join(dir, "config.json")
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(cfg.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0600 { // NTFS has no POSIX mode bits
		t.Errorf("config file with API keys must be 0600, got %04o", fi.Mode().Perm())
	}
}

func TestNewActiveProviderFactory(t *testing.T) {
	cfg := DefaultConfig()
	p, err := cfg.NewActiveProvider()
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	if p.Name() == "" {
		t.Error("provider has empty name")
	}
	if len(p.ListAvailableModels()) == 0 {
		t.Error("openrouter should expose quick-pick models")
	}
}

func TestMaxContextDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxContextTokens != DefaultMaxContextTokens {
		t.Errorf("expected default context budget %d, got %d", DefaultMaxContextTokens, cfg.MaxContextTokens)
	}
	if cfg.ApproveMode != "ask" {
		t.Errorf("expected approve mode ask, got %q", cfg.ApproveMode)
	}
}

func TestMaskKey(t *testing.T) {
	if MaskKey("") != "(not set)" {
		t.Error("empty key masking failed")
	}
	m := MaskKey("sk-proj-abcdefghijklmnop")
	if m == "sk-proj-abcdefghijklmnop" || len(m) > 14 {
		t.Errorf("key not masked: %s", m)
	}
}

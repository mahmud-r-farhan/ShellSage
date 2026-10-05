package provider

import (
	"testing"
)

func TestRegistryIntegrity(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range All() {
		if seen[d.ID] {
			t.Errorf("duplicate provider id: %s", d.ID)
		}
		seen[d.ID] = true

		if d.DisplayName == "" || d.Tagline == "" {
			t.Errorf("provider %s missing display metadata", d.ID)
		}
		if d.EnvPrefix == "" {
			t.Errorf("provider %s missing EnvPrefix (config env overrides depend on it)", d.ID)
		}
		if d.DefaultBaseURL == "" {
			t.Errorf("provider %s missing DefaultBaseURL", d.ID)
		}
		if d.DefaultModel == "" {
			t.Errorf("provider %s missing DefaultModel", d.ID)
		}
		switch d.Format {
		case FormatOpenAI, FormatAnthropic, FormatOllama:
		default:
			t.Errorf("provider %s has unknown format %q", d.ID, d.Format)
		}
		if d.DefaultModel != "default-model" && len(d.Models) > 0 {
			found := false
			for _, m := range d.Models {
				if m == d.DefaultModel {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("provider %s: DefaultModel %q not in quick-pick list %v", d.ID, d.DefaultModel, d.Models)
			}
		}
	}
}

func TestRegistryCoversLegacyProviders(t *testing.T) {
	required := []string{"openrouter", "openai", "anthropic", "gemini", "groq", "deepseek", "ollama", "custom"}
	for _, id := range required {
		if _, ok := ByID(id); !ok {
			t.Errorf("legacy provider %s missing from registry (v3 configs would break)", id)
		}
	}
}

func TestRegistryNewProvidersPresent(t *testing.T) {
	// The 12 providers introduced in v4.0.0 — asserted explicitly so future
	// refactors can't silently drop them.
	added := []string{"mistral", "xai", "together", "fireworks", "cerebras", "nvidia", "github", "huggingface", "perplexity", "deepinfra", "cohere", "azure"}
	for _, id := range added {
		d, ok := ByID(id)
		if !ok {
			t.Errorf("v4 provider %s missing", id)
			continue
		}
		if d.DisplayName == "" || d.Tagline == "" || d.DocsURL == "" {
			t.Errorf("provider %s missing display metadata", id)
		}
	}
}

func TestProviderAliases(t *testing.T) {
	for alias, want := range map[string]string{
		"claude":       "anthropic",
		"google":       "gemini",
		"gh":           "github",
		"grok":         "xai",
		"lmstudio":     "custom",
		"Azure-OpenAI": "azure",
	} {
		id, ok := NormalizeID(alias)
		if !ok || id != want {
			t.Errorf("alias %q resolved to %q (want %q)", alias, id, want)
		}
	}
}

func TestFactoryRoundTrip(t *testing.T) {
	for _, d := range All() {
		p, err := New(Selection{ID: d.ID, APIKey: "k"})
		if err != nil {
			t.Fatalf("New(%s): %v", d.ID, err)
		}
		if p == nil {
			t.Fatalf("New(%s) returned nil", d.ID)
		}
		if p.Name() == "" {
			t.Errorf("provider %s has empty Name()", d.ID)
		}
	}
}

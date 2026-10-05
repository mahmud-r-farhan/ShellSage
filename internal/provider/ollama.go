package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// OllamaProvider handles requests to a local Ollama instance
type OllamaProvider struct {
	*BaseOpenAICompatibleProvider
}

func NewOllamaProvider(baseURL, defaultModel string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
	}
	if defaultModel == "" {
		defaultModel = "llama3.2:latest"
	}
	base := NewBaseProvider("Ollama", "", baseURL, defaultModel, []string{defaultModel}, nil)
	return &OllamaProvider{BaseOpenAICompatibleProvider: base}
}

// FetchInstalledModels queries Ollama for the actual installed models
func (p *OllamaProvider) FetchInstalledModels() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	models, err := p.ListRemoteModels(ctx)
	if err != nil || len(models) == 0 {
		return []string{p.defaultModel}
	}
	return models
}

// ListAvailableModels prefers locally installed models so `/model` and
// `models list` show what the machine actually has (falls back to registry list).
func (p *OllamaProvider) ListAvailableModels() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if models, err := p.ListRemoteModels(ctx); err == nil && len(models) > 0 {
		return models
	}
	return p.BaseOpenAICompatibleProvider.ListAvailableModels()
}

// ListRemoteModels queries Ollama's /api/tags endpoint.
func (p *OllamaProvider) ListRemoteModels(ctx context.Context) ([]string, error) {
	endpoint := strings.TrimSuffix(p.baseURL, "/v1") + "/api/tags"
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot reach Ollama at %s (is `ollama serve` running?): %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama /api/tags returned status %d", resp.StatusCode)
	}

	var data struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode ollama tags: %w", err)
	}

	var names []string
	seen := map[string]bool{}
	for _, m := range data.Models {
		if m.Name != "" && !seen[m.Name] {
			seen[m.Name] = true
			names = append(names, m.Name)
		}
	}
	return names, nil
}

package provider

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// OllamaProvider handles requests to local Ollama instance
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
	endpoint := strings.TrimSuffix(p.baseURL, "/v1") + "/api/tags"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return []string{p.defaultModel}
	}
	defer resp.Body.Close()

	var data struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil || len(data.Models) == 0 {
		return []string{p.defaultModel}
	}

	var names []string
	for _, m := range data.Models {
		names = append(names, m.Name)
	}
	return names
}

package provider

// CustomProvider handles any custom OpenAI-compatible endpoint
type CustomProvider struct {
	*BaseOpenAICompatibleProvider
}

func NewCustomProvider(apiKey, baseURL, defaultModel string) *CustomProvider {
	if baseURL == "" {
		baseURL = "http://localhost:8080/v1"
	}
	if defaultModel == "" {
		defaultModel = "default-model"
	}
	base := NewBaseProvider("Custom", apiKey, baseURL, defaultModel, []string{defaultModel}, nil)
	return &CustomProvider{BaseOpenAICompatibleProvider: base}
}

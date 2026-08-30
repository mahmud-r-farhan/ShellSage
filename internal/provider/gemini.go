package provider

// GeminiProvider handles requests to Google Gemini OpenAI-compatible API
type GeminiProvider struct {
	*BaseOpenAICompatibleProvider
}

func NewGeminiProvider(apiKey, baseURL, defaultModel string) *GeminiProvider {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
	}
	if defaultModel == "" {
		defaultModel = "gemini-2.0-flash"
	}
	models := []string{
		"gemini-2.0-flash",
		"gemini-2.0-flash-thinking-exp-01-21",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
	}
	base := NewBaseProvider("Gemini", apiKey, baseURL, defaultModel, models, nil)
	return &GeminiProvider{BaseOpenAICompatibleProvider: base}
}

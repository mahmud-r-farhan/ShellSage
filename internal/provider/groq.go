package provider

// GroqProvider handles requests to Groq Cloud API
type GroqProvider struct {
	*BaseOpenAICompatibleProvider
}

func NewGroqProvider(apiKey, baseURL, defaultModel string) *GroqProvider {
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai/v1"
	}
	if defaultModel == "" {
		defaultModel = "llama-3.3-70b-versatile"
	}
	models := []string{
		"llama-3.3-70b-versatile",
		"llama-3.1-8b-instant",
		"mixtral-8x7b-32768",
		"deepseek-r1-distill-llama-70b",
		"gemma2-9b-it",
	}
	base := NewBaseProvider("Groq", apiKey, baseURL, defaultModel, models, nil)
	return &GroqProvider{BaseOpenAICompatibleProvider: base}
}

package provider

// OpenAIProvider handles requests to the official OpenAI API
type OpenAIProvider struct {
	*BaseOpenAICompatibleProvider
}

func NewOpenAIProvider(apiKey, baseURL, defaultModel string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if defaultModel == "" {
		defaultModel = "gpt-4o-mini"
	}
	models := []string{
		"gpt-4o",
		"gpt-4o-mini",
		"o1",
		"o1-mini",
		"o3-mini",
		"gpt-4-turbo",
	}
	base := NewBaseProvider("OpenAI", apiKey, baseURL, defaultModel, models, nil)
	return &OpenAIProvider{BaseOpenAICompatibleProvider: base}
}

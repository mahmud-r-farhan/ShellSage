package provider

// DeepSeekProvider handles requests to DeepSeek API
type DeepSeekProvider struct {
	*BaseOpenAICompatibleProvider
}

func NewDeepSeekProvider(apiKey, baseURL, defaultModel string) *DeepSeekProvider {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}
	if defaultModel == "" {
		defaultModel = "deepseek-chat"
	}
	models := []string{
		"deepseek-chat",
		"deepseek-reasoner",
	}
	base := NewBaseProvider("DeepSeek", apiKey, baseURL, defaultModel, models, nil)
	return &DeepSeekProvider{BaseOpenAICompatibleProvider: base}
}

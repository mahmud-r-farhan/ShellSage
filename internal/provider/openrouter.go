package provider

// OpenRouterProvider handles requests to OpenRouter API
type OpenRouterProvider struct {
	*BaseOpenAICompatibleProvider
}

func NewOpenRouterProvider(apiKey, baseURL, defaultModel string) *OpenRouterProvider {
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	if defaultModel == "" {
		defaultModel = "openrouter/free"
	}
	models := []string{
		"openrouter/free",
		"meta-llama/llama-3.3-70b-instruct",
		"anthropic/claude-3.5-sonnet",
		"google/gemini-2.0-flash-001",
		"openai/gpt-4o-mini",
		"deepseek/deepseek-r1",
		"mistralai/mistral-large-2411",
		"qwen/qwen-2.5-coder-32b-instruct",
	}
	extraHeaders := map[string]string{
		"HTTP-Referer": "https://github.com/mahmud-r-farhan/ShellSage",
		"X-Title":      "ShellSage CLI",
	}
	base := NewBaseProvider("OpenRouter", apiKey, baseURL, defaultModel, models, extraHeaders)
	return &OpenRouterProvider{BaseOpenAICompatibleProvider: base}
}

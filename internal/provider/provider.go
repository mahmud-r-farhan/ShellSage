package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"shellsage/internal/config"
)

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents the unified request payload
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream"`
}

// TokenUsage holds token consumption metadata
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse represents the unified response payload
type ChatResponse struct {
	ID           string     `json:"id"`
	Model        string     `json:"model"`
	Content      string     `json:"content"`
	Usage        TokenUsage `json:"usage"`
	FinishReason string     `json:"finish_reason"`
}

// StreamCallback is called for every incoming token delta
type StreamCallback func(chunk string) error

// Provider is the common interface implemented by all LLM adapters
type Provider interface {
	Name() string
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	Stream(ctx context.Context, req *ChatRequest, callback StreamCallback) (*ChatResponse, error)
	ListAvailableModels() []string
}

// BaseOpenAICompatibleProvider implements common logic for OpenAI-like endpoints
type BaseOpenAICompatibleProvider struct {
	providerName string
	apiKey       string
	baseURL      string
	defaultModel string
	httpClient   *http.Client
	models       []string
	extraHeaders map[string]string
}

// NewBaseProvider creates a reusable base provider
func NewBaseProvider(name, apiKey, baseURL, defaultModel string, models []string, extraHeaders map[string]string) *BaseOpenAICompatibleProvider {
	return &BaseOpenAICompatibleProvider{
		providerName: name,
		apiKey:       apiKey,
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		defaultModel: defaultModel,
		httpClient: &http.Client{
			Timeout: 180 * time.Second,
		},
		models:       models,
		extraHeaders: extraHeaders,
	}
}

func (p *BaseOpenAICompatibleProvider) Name() string {
	return p.providerName
}

func (p *BaseOpenAICompatibleProvider) ListAvailableModels() []string {
	return p.models
}

func (p *BaseOpenAICompatibleProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = p.defaultModel
	}
	req.Stream = false
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	endpoint := p.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error connecting to %s API: %w", p.providerName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s API error (status %d): %s", p.providerName, resp.StatusCode, string(respBytes))
	}

	var rawResp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message      Message `json:"message"`
			FinishReason string  `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if len(rawResp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned from %s", p.providerName)
	}

	// Fallback token estimation if API didn't return usage
	usage := TokenUsage{
		PromptTokens:     rawResp.Usage.PromptTokens,
		CompletionTokens: rawResp.Usage.CompletionTokens,
		TotalTokens:      rawResp.Usage.TotalTokens,
	}
	if usage.TotalTokens == 0 {
		usage = EstimateUsage(req.Messages, rawResp.Choices[0].Message.Content)
	}

	return &ChatResponse{
		ID:           rawResp.ID,
		Model:        rawResp.Model,
		Content:      rawResp.Choices[0].Message.Content,
		Usage:        usage,
		FinishReason: rawResp.Choices[0].FinishReason,
	}, nil
}

func (p *BaseOpenAICompatibleProvider) Stream(ctx context.Context, req *ChatRequest, callback StreamCallback) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = p.defaultModel
	}
	req.Stream = true
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	endpoint := p.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	p.setHeaders(httpReq)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error connecting to %s API stream: %w", p.providerName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s API streaming error (status %d): %s", p.providerName, resp.StatusCode, string(respBytes))
	}

	scanner := bufio.NewScanner(resp.Body)
	var fullContent strings.Builder
	var lastModel string
	var usage TokenUsage

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if strings.TrimSpace(data) == "[DONE]" {
			break
		}

		var chunk struct {
			Model   string `json:"model"`
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if chunk.Model != "" {
			lastModel = chunk.Model
		}

		if chunk.Usage != nil && chunk.Usage.TotalTokens > 0 {
			usage = TokenUsage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
		}

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta.Content
			if delta != "" {
				fullContent.WriteString(delta)
				if callback != nil {
					if err := callback(delta); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return nil, fmt.Errorf("error reading stream from %s: %w", p.providerName, err)
	}

	content := fullContent.String()
	if usage.TotalTokens == 0 {
		usage = EstimateUsage(req.Messages, content)
	}

	return &ChatResponse{
		Model:   lastModel,
		Content: content,
		Usage:   usage,
	}, nil
}

func (p *BaseOpenAICompatibleProvider) setHeaders(httpReq *http.Request) {
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	httpReq.Header.Set("User-Agent", "ShellSage/3.0")
	for k, v := range p.extraHeaders {
		httpReq.Header.Set(k, v)
	}
}

// EstimateUsage provides rough token counts when providers don't report them
func EstimateUsage(messages []Message, response string) TokenUsage {
	promptChars := 0
	for _, m := range messages {
		promptChars += len(m.Content) + len(m.Role)
	}
	promptTokens := promptChars / 4
	if promptTokens < 1 {
		promptTokens = 1
	}

	completionTokens := len(response) / 4
	if completionTokens < 1 && len(response) > 0 {
		completionTokens = 1
	}

	return TokenUsage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}
}

// Factory to instantiate the active provider
func NewProvider(cfg *config.Config) (Provider, error) {
	pType := cfg.ActiveProvider
	pCfg, ok := cfg.Providers[pType]
	if !ok {
		return nil, fmt.Errorf("provider configuration not found for %s", pType)
	}

	switch pType {
	case config.ProviderOpenRouter:
		return NewOpenRouterProvider(pCfg.APIKey, pCfg.BaseURL, pCfg.Model), nil
	case config.ProviderOpenAI:
		return NewOpenAIProvider(pCfg.APIKey, pCfg.BaseURL, pCfg.Model), nil
	case config.ProviderAnthropic:
		return NewAnthropicProvider(pCfg.APIKey, pCfg.BaseURL, pCfg.Model), nil
	case config.ProviderGemini:
		return NewGeminiProvider(pCfg.APIKey, pCfg.BaseURL, pCfg.Model), nil
	case config.ProviderGroq:
		return NewGroqProvider(pCfg.APIKey, pCfg.BaseURL, pCfg.Model), nil
	case config.ProviderDeepSeek:
		return NewDeepSeekProvider(pCfg.APIKey, pCfg.BaseURL, pCfg.Model), nil
	case config.ProviderOllama:
		return NewOllamaProvider(pCfg.BaseURL, pCfg.Model), nil
	case config.ProviderCustom:
		return NewCustomProvider(pCfg.APIKey, pCfg.BaseURL, pCfg.Model), nil
	default:
		return NewOpenRouterProvider(pCfg.APIKey, pCfg.BaseURL, pCfg.Model), nil
	}
}

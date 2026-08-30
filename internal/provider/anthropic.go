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
)

// AnthropicProvider handles requests to Anthropic Claude Messages API
type AnthropicProvider struct {
	apiKey       string
	baseURL      string
	defaultModel string
	httpClient   *http.Client
	models       []string
}

func NewAnthropicProvider(apiKey, baseURL, defaultModel string) *AnthropicProvider {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	if defaultModel == "" {
		defaultModel = "claude-3-5-haiku-latest"
	}
	models := []string{
		"claude-3-7-sonnet-latest",
		"claude-3-5-sonnet-latest",
		"claude-3-5-haiku-latest",
		"claude-3-opus-latest",
	}
	return &AnthropicProvider{
		apiKey:       apiKey,
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		defaultModel: defaultModel,
		httpClient: &http.Client{
			Timeout: 180 * time.Second,
		},
		models: models,
	}
}

func (p *AnthropicProvider) Name() string {
	return "Anthropic"
}

func (p *AnthropicProvider) ListAvailableModels() []string {
	return p.models
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicReq struct {
	Model       string         `json:"model"`
	MaxTokens   int            `json:"max_tokens"`
	System      string         `json:"system,omitempty"`
	Messages    []anthropicMsg `json:"messages"`
	Temperature float64        `json:"temperature,omitempty"`
	Stream      bool           `json:"stream,omitempty"`
}

func (p *AnthropicProvider) formatRequest(req *ChatRequest) *anthropicReq {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	var systemPrompt string
	var messages []anthropicMsg

	for _, m := range req.Messages {
		if m.Role == "system" {
			if systemPrompt != "" {
				systemPrompt += "\n"
			}
			systemPrompt += m.Content
		} else {
			messages = append(messages, anthropicMsg{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	}

	return &anthropicReq{
		Model:       model,
		MaxTokens:   maxTokens,
		System:      systemPrompt,
		Messages:    messages,
		Temperature: req.Temperature,
		Stream:      req.Stream,
	}
}

func (p *AnthropicProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	req.Stream = false
	aReq := p.formatRequest(req)

	body, err := json.Marshal(aReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling anthropic request: %w", err)
	}

	endpoint := p.baseURL + "/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error creating anthropic request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("User-Agent", "ShellSage/3.0")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error connecting to Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Anthropic API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var rawResp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		StopReason string `json:"stop_reason"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
		return nil, fmt.Errorf("error decoding Anthropic response: %w", err)
	}

	var fullText strings.Builder
	for _, c := range rawResp.Content {
		if c.Type == "text" {
			fullText.WriteString(c.Text)
		}
	}

	return &ChatResponse{
		ID:      rawResp.ID,
		Model:   rawResp.Model,
		Content: fullText.String(),
		Usage: TokenUsage{
			PromptTokens:     rawResp.Usage.InputTokens,
			CompletionTokens: rawResp.Usage.OutputTokens,
			TotalTokens:      rawResp.Usage.InputTokens + rawResp.Usage.OutputTokens,
		},
		FinishReason: rawResp.StopReason,
	}, nil
}

func (p *AnthropicProvider) Stream(ctx context.Context, req *ChatRequest, callback StreamCallback) (*ChatResponse, error) {
	req.Stream = true
	aReq := p.formatRequest(req)

	body, err := json.Marshal(aReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling anthropic request: %w", err)
	}

	endpoint := p.baseURL + "/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error creating anthropic request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("User-Agent", "ShellSage/3.0")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error connecting to Anthropic API stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Anthropic streaming error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	scanner := bufio.NewScanner(resp.Body)
	var fullText strings.Builder
	var promptTokens, completionTokens int

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
			Message struct {
				Usage struct {
					InputTokens int `json:"input_tokens"`
				} `json:"usage"`
			} `json:"message"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if event.Type == "message_start" {
			promptTokens = event.Message.Usage.InputTokens
		} else if event.Type == "content_block_delta" && event.Delta.Text != "" {
			fullText.WriteString(event.Delta.Text)
			if callback != nil {
				if err := callback(event.Delta.Text); err != nil {
					return nil, err
				}
			}
		} else if event.Type == "message_delta" {
			completionTokens = event.Usage.OutputTokens
		}
	}

	content := fullText.String()
	usage := TokenUsage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}
	if usage.TotalTokens == 0 {
		usage = EstimateUsage(req.Messages, content)
	}

	return &ChatResponse{
		Model:   aReq.Model,
		Content: content,
		Usage:   usage,
	}, nil
}

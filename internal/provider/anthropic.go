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

// AnthropicProvider handles requests to the Anthropic Claude Messages API
type AnthropicProvider struct {
	apiKey       string
	baseURL      string
	defaultModel string
	httpClient   *http.Client
	models       []string
	maxRetries   int
}

func NewAnthropicProvider(apiKey, baseURL, defaultModel string) *AnthropicProvider {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	if defaultModel == "" {
		defaultModel = "claude-3-5-haiku-latest"
	}
	models := []string{
		"claude-sonnet-4-20250514",
		"claude-opus-4-20250514",
		"claude-3-7-sonnet-latest",
		"claude-3-5-sonnet-latest",
		"claude-3-5-haiku-latest",
	}
	return &AnthropicProvider{
		apiKey:       apiKey,
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		defaultModel: defaultModel,
		httpClient: &http.Client{
			Timeout: 180 * time.Second,
		},
		models:     models,
		maxRetries: defaultMaxRetries,
	}
}

func (p *AnthropicProvider) Name() string {
	return "Anthropic"
}

func (p *AnthropicProvider) ListAvailableModels() []string {
	return p.models
}

// SetMaxRetries overrides the retry budget for transient failures.
func (p *AnthropicProvider) SetMaxRetries(n int) { p.maxRetries = n }

// SetTimeout adjusts the overall HTTP client timeout.
func (p *AnthropicProvider) SetTimeout(d time.Duration) {
	if d > 0 {
		p.httpClient.Timeout = d
	}
}

type anthropicContentPart struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	Source *struct {
		Type      string `json:"type"`
		MediaType string `json:"media_type,omitempty"`
		Data      string `json:"data,omitempty"`
		URL       string `json:"url,omitempty"`
	} `json:"source,omitempty"`
}

type anthropicMsg struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type anthropicReq struct {
	Model       string         `json:"model"`
	MaxTokens   int            `json:"max_tokens"`
	System      string         `json:"system,omitempty"`
	Messages    []anthropicMsg `json:"messages"`
	Temperature float64        `json:"temperature,omitempty"`
	Stream      bool           `json:"stream,omitempty"`
}

// parseDataURI splits "data:<media>;base64,<payload>"; returns ok=false for URLs.
func parseDataURI(s string) (media, data string, ok bool) {
	if !strings.HasPrefix(s, "data:") {
		return "", "", false
	}
	semicolon := strings.Index(s, ";")
	comma := strings.Index(s, ",")
	if semicolon < 0 || comma < semicolon {
		return "", "", false
	}
	return s[len("data:"):semicolon], s[comma+1:], true
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
			continue
		}

		if len(m.Images) == 0 {
			messages = append(messages, anthropicMsg{Role: m.Role, Content: m.Content})
			continue
		}

		parts := []anthropicContentPart{{Type: "text", Text: m.Content}}
		for _, img := range m.Images {
			if media, data, ok := parseDataURI(img); ok {
				part := anthropicContentPart{Type: "image"}
				part.Source = &struct {
					Type      string `json:"type"`
					MediaType string `json:"media_type,omitempty"`
					Data      string `json:"data,omitempty"`
					URL       string `json:"url,omitempty"`
				}{Type: "base64", MediaType: media, Data: data}
				parts = append(parts, part)
			} else {
				part := anthropicContentPart{Type: "image"}
				part.Source = &struct {
					Type      string `json:"type"`
					MediaType string `json:"media_type,omitempty"`
					Data      string `json:"data,omitempty"`
					URL       string `json:"url,omitempty"`
				}{Type: "url", URL: img}
				parts = append(parts, part)
			}
		}
		messages = append(messages, anthropicMsg{Role: m.Role, Content: parts})
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

func (p *AnthropicProvider) setHeaders(httpReq *http.Request) {
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("User-Agent", "ShellSage/4.0")
}

func (p *AnthropicProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	req.Stream = false
	aReq := p.formatRequest(req)

	body, err := json.Marshal(aReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling anthropic request: %w", err)
	}

	endpoint := p.baseURL + "/messages"
	resp, err := doWithRetry(ctx, p.httpClient, p.maxRetries, func() (*http.Request, error) {
		httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
		if err != nil {
			return nil, fmt.Errorf("error creating anthropic request: %w", err)
		}
		p.setHeaders(httpReq)
		return httpReq, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error connecting to Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, formatAPIError("Anthropic", resp)
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
	resp, err := doWithRetry(ctx, p.httpClient, p.maxRetries, func() (*http.Request, error) {
		httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
		if err != nil {
			return nil, fmt.Errorf("error creating anthropic request: %w", err)
		}
		p.setHeaders(httpReq)
		httpReq.Header.Set("Accept", "text/event-stream")
		return httpReq, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error connecting to Anthropic API stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, formatAPIError("Anthropic", resp)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
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

// ListRemoteModels queries Anthropic's /v1/models endpoint.
func (p *AnthropicProvider) ListRemoteModels(ctx context.Context) ([]string, error) {
	endpoint := p.baseURL + "/models"
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	p.setHeaders(req)
	req.Header.Del("Content-Type")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot reach Anthropic models API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("anthropic /models status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode anthropic models: %w", err)
	}

	var names []string
	seen := map[string]bool{}
	for _, m := range out.Data {
		if m.ID != "" && !seen[m.ID] {
			seen[m.ID] = true
			names = append(names, m.ID)
		}
	}
	return names, nil
}

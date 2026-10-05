package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Message represents a chat message. Images (optional) carries multimodal
// attachments as data URIs ("data:image/png;base64,...") or http(s) URLs;
// OpenAI-compatible and Anthropic providers render them as content parts.
type Message struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"`
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

// RemoteModelLister is implemented by providers that can enumerate live models
// from their API (`/models`, or Ollama's `/api/tags`).
type RemoteModelLister interface {
	ListRemoteModels(ctx context.Context) ([]string, error)
}

// ModelFetcher is implemented by providers able to query locally installed
// models (e.g. Ollama).
type ModelFetcher interface {
	FetchInstalledModels() []string
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
	authStyle    string // "" => Authorization: Bearer, "api-key" => api-key header
	maxRetries   int
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
		maxRetries:   defaultMaxRetries,
	}
}

func (p *BaseOpenAICompatibleProvider) Name() string {
	return p.providerName
}

func (p *BaseOpenAICompatibleProvider) ListAvailableModels() []string {
	return p.models
}

// SetAuthStyle configures the API-key transport header ("api-key" for Azure).
func (p *BaseOpenAICompatibleProvider) SetAuthStyle(style string) { p.authStyle = style }

// SetMaxRetries overrides the retry budget for transient failures.
func (p *BaseOpenAICompatibleProvider) SetMaxRetries(n int) { p.maxRetries = n }

// SetTimeout adjusts the overall HTTP client timeout.
func (p *BaseOpenAICompatibleProvider) SetTimeout(d time.Duration) {
	if d > 0 {
		p.httpClient.Timeout = d
	}
}

func (p *BaseOpenAICompatibleProvider) setHeaders(httpReq *http.Request) {
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		switch p.authStyle {
		case "api-key":
			httpReq.Header.Set("api-key", p.apiKey)
		default:
			httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
		}
	}
	httpReq.Header.Set("User-Agent", "ShellSage/4.0")
	for k, v := range p.extraHeaders {
		httpReq.Header.Set(k, v)
	}
}

// buildPayload serializes a ChatRequest, converting messages with attached
// images into OpenAI multimodal `content` part arrays.
func (p *BaseOpenAICompatibleProvider) buildPayload(req *ChatRequest, stream bool) ([]byte, error) {
	type imageURL struct {
		URL string `json:"url"`
	}
	type part struct {
		Type     string    `json:"type"`
		Text     string    `json:"text,omitempty"`
		ImageURL *imageURL `json:"image_url,omitempty"`
	}
	type wireMsg struct {
		Role    string      `json:"role"`
		Content interface{} `json:"content"`
	}
	type wireReq struct {
		Model       string    `json:"model"`
		Messages    []wireMsg `json:"messages"`
		Temperature float64   `json:"temperature,omitempty"`
		MaxTokens   int       `json:"max_tokens,omitempty"`
		Stream      bool      `json:"stream"`
	}

	anyImages := false
	for _, m := range req.Messages {
		if len(m.Images) > 0 {
			anyImages = true
			break
		}
	}

	wr := wireReq{
		Model:       req.Model,
		Messages:    make([]wireMsg, 0, len(req.Messages)),
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      stream,
	}

	for _, m := range req.Messages {
		if !anyImages || len(m.Images) == 0 {
			wr.Messages = append(wr.Messages, wireMsg{Role: m.Role, Content: m.Content})
			continue
		}
		parts := []part{{Type: "text", Text: m.Content}}
		for _, img := range m.Images {
			parts = append(parts, part{Type: "image_url", ImageURL: &imageURL{URL: img}})
		}
		wr.Messages = append(wr.Messages, wireMsg{Role: m.Role, Content: parts})
	}

	return json.Marshal(wr)
}

func (p *BaseOpenAICompatibleProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = p.defaultModel
	}
	req.Stream = false
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}

	body, err := p.buildPayload(req, false)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	endpoint := p.baseURL + "/chat/completions"
	resp, err := doWithRetry(ctx, p.httpClient, p.maxRetries, func() (*http.Request, error) {
		httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}
		p.setHeaders(httpReq)
		return httpReq, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error connecting to %s API: %w", p.providerName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, formatAPIError(p.providerName, resp)
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

	body, err := p.buildPayload(req, true)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	endpoint := p.baseURL + "/chat/completions"
	// Retries apply to establishing the stream; mid-stream breaks surface as errors.
	resp, err := doWithRetry(ctx, p.httpClient, p.maxRetries, func() (*http.Request, error) {
		httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}
		p.setHeaders(httpReq)
		httpReq.Header.Set("Accept", "text/event-stream")
		return httpReq, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error connecting to %s API stream: %w", p.providerName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, formatAPIError(p.providerName, resp)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // long SSE lines on big chunks
	var fullContent strings.Builder
	var lastModel string
	var usage TokenUsage
	var finishReason string

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
			if chunk.Choices[0].FinishReason != "" {
				finishReason = chunk.Choices[0].FinishReason
			}
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
		Model:        lastModel,
		Content:      content,
		Usage:        usage,
		FinishReason: finishReason,
	}, nil
}

// ListRemoteModels queries the provider's `/models` endpoint (OpenAI standard).
func (p *BaseOpenAICompatibleProvider) ListRemoteModels(ctx context.Context) ([]string, error) {
	endpoint := p.baseURL + "/models"
	client := &http.Client{Timeout: 15 * time.Second}

	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	p.setHeaders(req)
	req.Header.Del("Content-Type")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: cannot reach %s: %w", p.providerName, endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("%s /models returned status %d: %s", p.providerName, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("%s: failed to decode /models response: %w", p.providerName, err)
	}
	if out.Error.Message != "" {
		return nil, fmt.Errorf("%s: %s", p.providerName, out.Error.Message)
	}

	seen := map[string]bool{}
	var names []string
	for _, m := range out.Data {
		if m.ID != "" && !seen[m.ID] {
			seen[m.ID] = true
			names = append(names, m.ID)
		}
	}
	return names, nil
}

// EstimateUsage provides rough token counts when providers don't report them.
// It excludes image data URIs from the character count so costs stay sane.
func EstimateUsage(messages []Message, response string) TokenUsage {
	promptChars := 0
	for _, m := range messages {
		promptChars += len(m.Content) + len(m.Role)
		if len(m.Images) > 0 {
			// Heuristic: each attached image ≈ 850 tokens for vision models.
			promptChars += len(m.Images) * 850 * 4
		}
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

// LoadImageAsDataURI reads a local image file (png/jpg/jpeg/gif/webp) or passes
// through an http(s) URL / data URI unchanged. Used by --image and /image.
func LoadImageAsDataURI(pathOrURL string) (string, error) {
	s := strings.TrimSpace(pathOrURL)
	if s == "" {
		return "", fmt.Errorf("empty image reference")
	}
	if strings.HasPrefix(s, "data:image/") {
		return s, nil
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return s, nil
	}

	data, err := os.ReadFile(s)
	if err != nil {
		// Also support `file://` style paths stripped of scheme.
		data, err = os.ReadFile(strings.TrimPrefix(s, "file://"))
		if err != nil {
			return "", fmt.Errorf("cannot read image '%s': %w", s, err)
		}
	}

	mime := "image/png"
	switch strings.ToLower(filepath.Ext(s)) {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".gif":
		mime = "image/gif"
	case ".webp":
		mime = "image/webp"
	}
	if len(data) > 8*1024*1024 {
		return "", fmt.Errorf("image '%s' exceeds 8MB inline limit", s)
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

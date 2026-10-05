package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestOpenAICompatibleProviderChat(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing or invalid authorization header")
		}

		resp := map[string]interface{}{
			"id":    "chatcmpl-123",
			"model": "gpt-4o-mini",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "Hello from mock provider!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	p, err := New(Selection{ID: "openai", APIKey: "test-key", BaseURL: mockServer.URL, Model: "gpt-4o-mini"})
	if err != nil {
		t.Fatalf("factory error: %v", err)
	}
	resp, err := p.Chat(context.Background(), &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "Hi"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "Hello from mock provider!" {
		t.Errorf("unexpected response content: %s", resp.Content)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("unexpected total tokens: %d", resp.Usage.TotalTokens)
	}
}

func TestGroqProviderChatAndStream(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer gsk-test-key" {
			t.Errorf("missing or invalid Groq authorization header")
		}

		if r.Header.Get("Accept") == "text/event-stream" {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"model\":\"llama-3.3-70b-versatile\",\"choices\":[{\"delta\":{\"content\":\"Groq \"}}]}\n\n"))
			_, _ = w.Write([]byte("data: {\"model\":\"llama-3.3-70b-versatile\",\"choices\":[{\"delta\":{\"content\":\"fast!\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":12,\"completion_tokens\":2,\"total_tokens\":14}}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}

		resp := map[string]interface{}{
			"id":    "chatcmpl-groq-123",
			"model": "llama-3.3-70b-versatile",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "Hello from Groq Cloud!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	p, err := New(Selection{ID: "groq", APIKey: "gsk-test-key", BaseURL: mockServer.URL, Model: "llama-3.3-70b-versatile"})
	if err != nil {
		t.Fatalf("factory error for Groq: %v", err)
	}
	if p.Name() != "Groq Cloud" {
		t.Errorf("expected provider name 'Groq Cloud', got %q", p.Name())
	}

	// Test Chat
	resp, err := p.Chat(context.Background(), &ChatRequest{
		Messages: []Message{{Role: "user", Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("unexpected Groq Chat error: %v", err)
	}
	if resp.Content != "Hello from Groq Cloud!" {
		t.Errorf("unexpected Groq content: %s", resp.Content)
	}

	// Test Stream
	var streamBuf string
	streamResp, err := p.Stream(context.Background(), &ChatRequest{
		Messages: []Message{{Role: "user", Content: "Stream test"}},
	}, func(chunk string) error {
		streamBuf += chunk
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected Groq Stream error: %v", err)
	}
	if streamBuf != "Groq fast!" {
		t.Errorf("expected streamed buffer 'Groq fast!', got %q", streamBuf)
	}
	if streamResp.Usage.TotalTokens != 14 {
		t.Errorf("expected total tokens 14, got %d", streamResp.Usage.TotalTokens)
	}
}

func TestEstimateUsage(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "Hello world this is a test"},
	}
	response := "This is a simple answer from the assistant."

	usage := EstimateUsage(messages, response)
	if usage.PromptTokens == 0 || usage.CompletionTokens == 0 || usage.TotalTokens == 0 {
		t.Errorf("token estimation failed, got %+v", usage)
	}
}

func TestMultimodalRequestPayload(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "ok"}, "finish_reason": "stop"},
			},
		})
	}))
	defer srv.Close()

	base := NewBaseProvider("Test", "k", srv.URL, "m", nil, nil)
	_, err := base.Chat(context.Background(), &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "what is in this image?", Images: []string{"data:image/png;base64,AAAA"}},
		},
	})
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	msgs, _ := gotBody["messages"].([]interface{})
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	content, _ := msgs[0].(map[string]interface{})["content"]
	parts, ok := content.([]interface{})
	if !ok || len(parts) != 2 {
		t.Fatalf("expected multimodal content parts, got %#v", content)
	}
	first := parts[0].(map[string]interface{})
	if first["type"] != "text" || first["text"] != "what is in this image?" {
		t.Errorf("unexpected text part: %#v", first)
	}
	second := parts[1].(map[string]interface{})
	iu, _ := second["image_url"].(map[string]interface{})
	if iu == nil || iu["url"] != "data:image/png;base64,AAAA" {
		t.Errorf("unexpected image part: %#v", second)
	}
}

func TestRetryOn429ThenSuccess(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "recovered"}, "finish_reason": "stop"},
			},
		})
	}))
	defer srv.Close()

	base := NewBaseProvider("Test", "k", srv.URL, "m", nil, nil)
	base.httpClient = srv.Client() // keep same transport; timeout still client-managed

	done := make(chan *ChatResponse, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := base.Chat(context.Background(), &ChatRequest{Messages: []Message{{Role: "user", Content: "hi"}}})
		if err != nil {
			errCh <- err
			return
		}
		done <- resp
	}()

	select {
	case resp := <-done:
		if resp.Content != "recovered" {
			t.Errorf("expected recovered content, got %q", resp.Content)
		}
		if got := atomic.LoadInt32(&hits); got != 3 {
			t.Errorf("expected 3 attempts, got %d", got)
		}
	case err := <-errCh:
		t.Fatalf("retry did not recover: %v", err)
	case <-time.After(20 * time.Second):
		t.Fatal("timed out waiting for retries")
	}
}

func TestAzureAuthStyleUsesAPIKeyHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("api-key")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "hi"}, "finish_reason": "stop"},
			},
		})
	}))
	defer srv.Close()

	base := NewBaseProvider("Azure", "secret-key", srv.URL, "dep1", nil, nil)
	base.SetAuthStyle("api-key")
	if _, err := base.Chat(context.Background(), &ChatRequest{Messages: []Message{{Role: "user", Content: "x"}}}); err != nil {
		t.Fatalf("chat: %v", err)
	}
	if got != "secret-key" {
		t.Errorf("expected api-key header, got %q", got)
	}
}

func TestUnknownProviderError(t *testing.T) {
	_, err := New(Selection{ID: "nope"})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if _, ok := err.(*UnknownProviderError); !ok {
		t.Errorf("expected *UnknownProviderError, got %T: %v", err, err)
	}
}

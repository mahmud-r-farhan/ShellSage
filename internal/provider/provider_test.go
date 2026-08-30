package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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

	p := NewOpenAIProvider("test-key", mockServer.URL, "gpt-4o-mini")
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

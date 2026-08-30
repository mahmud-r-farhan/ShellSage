package agent

import (
	"context"
	"strings"
	"testing"

	"shellsage/internal/provider"
	"shellsage/internal/tools"
)

type mockProvider struct {
	responses []string
	callIdx   int
}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) ListAvailableModels() []string { return []string{"mock-model"} }
func (m *mockProvider) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	resp := m.responses[m.callIdx]
	m.callIdx++
	return &provider.ChatResponse{
		Content: resp,
		Usage:   provider.TokenUsage{TotalTokens: 10},
	}, nil
}
func (m *mockProvider) Stream(ctx context.Context, req *provider.ChatRequest, cb provider.StreamCallback) (*provider.ChatResponse, error) {
	return m.Chat(ctx, req)
}

func TestAgentReActLoop(t *testing.T) {
	mockP := &mockProvider{
		responses: []string{
			"I will list the directory.\n```tool_call\n{\"tool\": \"list_dir\", \"arguments\": {\"path\": \".\"}}\n```",
			"Here is the final answer: Found files in the directory.",
		},
	}

	reg := tools.NewToolRegistry()
	ag := NewAgent(mockP, reg, "mock-model", 0.1)

	var toolsCalled []string
	listener := &DefaultListener{
		ToolCallHandler: func(name, args string) {
			toolsCalled = append(toolsCalled, name)
		},
	}

	ans, err := ag.Run(context.Background(), "Check current files", listener)
	if err != nil {
		t.Fatalf("unexpected agent error: %v", err)
	}

	if len(toolsCalled) != 1 || toolsCalled[0] != "list_dir" {
		t.Errorf("expected tool call 'list_dir', got %v", toolsCalled)
	}

	if !strings.Contains(ans, "Found files in the directory") {
		t.Errorf("unexpected final answer: %s", ans)
	}
}

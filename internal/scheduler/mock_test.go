package scheduler

import (
	"context"
	"testing"

	"shellsage/internal/agent"
	"shellsage/internal/provider"
	"shellsage/internal/tools"
)

type mockProviderV4 struct{}

func (m *mockProviderV4) Name() string                  { return "mock" }
func (m *mockProviderV4) ListAvailableModels() []string { return []string{"m"} }
func (m *mockProviderV4) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{Content: "ok", Usage: provider.TokenUsage{TotalTokens: 1}}, nil
}
func (m *mockProviderV4) Stream(ctx context.Context, req *provider.ChatRequest, cb provider.StreamCallback) (*provider.ChatResponse, error) {
	return m.Chat(ctx, req)
}

func agentForMock(t *testing.T) *agent.Agent {
	return agent.NewAgent(&mockProviderV4{}, tools.NewToolRegistry(), "m", 0.1)
}

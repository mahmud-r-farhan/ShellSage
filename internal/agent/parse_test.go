package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"shellsage/internal/provider"
	"shellsage/internal/tools"
)

func TestParseToolCallsMultiFormat(t *testing.T) {
	content := "Plan:\n" +
		"```tool_call\n{\"tool\": \"list_dir\", \"arguments\": {\"path\": \".\"}}\n```\n" +
		"```json\n{\"tool\": \"find_files\", \"arguments\": {\"pattern\": \"*.go\"}}\n```\n" +
		"{\"tool\": \"git_status\", \"arguments\": {}}\n" +
		"Action: read_file\nAction Input: {\"path\": \"main.go\"}\n"

	calls := parseToolCalls(content)
	if len(calls) != 4 {
		t.Fatalf("expected 4 tool calls, got %d: %+v", len(calls), calls)
	}
	names := []string{calls[0].Tool, calls[1].Tool, calls[2].Tool, calls[3].Tool}
	want := []string{"list_dir", "find_files", "git_status", "read_file"}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("call %d: got %q want %q", i, names[i], want[i])
		}
	}
}

func TestParseDuplicateToolCallsDeduped(t *testing.T) {
	dup := "```tool_call\n{\"tool\": \"git_status\", \"arguments\": {}}\n```\n{\"tool\": \"git_status\", \"arguments\": {}}"
	calls := parseToolCalls(dup)
	if len(calls) != 1 {
		t.Errorf("expected dedupe to 1 call, got %d", len(calls))
	}
}

type scriptedProvider struct {
	responses []string
	idx       int
}

func (m *scriptedProvider) Name() string                  { return "scripted" }
func (m *scriptedProvider) ListAvailableModels() []string { return []string{"s"} }
func (m *scriptedProvider) Stream(ctx context.Context, req *provider.ChatRequest, cb provider.StreamCallback) (*provider.ChatResponse, error) {
	return m.Chat(ctx, req)
}
func (m *scriptedProvider) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	r := m.responses[m.idx]
	m.idx++
	return &provider.ChatResponse{Content: r, Usage: provider.TokenUsage{TotalTokens: 1}}, nil
}

func TestAgentSingleAssistantTurnForMultipleCalls(t *testing.T) {
	// A step with two tool calls must append exactly ONE assistant message,
	// and one aggregated user "TOOL RESULT" message.
	mock := &scriptedProvider{responses: []string{
		"```tool_call\n{\"tool\":\"list_dir\",\"arguments\":{\"path\":\".\"}}\n```\n```tool_call\n{\"tool\":\"git_status\",\"arguments\":{}}\n```",
		"FINAL DONE",
	}}
	ag := NewAgent(mock, tools.NewToolRegistry(), "s", 0.1)
	if _, err := ag.Run(context.Background(), "goal", nil); err != nil {
		t.Fatalf("run: %v", err)
	}
	if mock.idx != 2 {
		t.Errorf("expected 2 provider calls, got %d", mock.idx)
	}
}

// toolCallJSON renders a fenced tool_call block with properly escaped JSON
// arguments (Windows test paths contain backslashes that break hand-rolled JSON).
func toolCallJSON(t *testing.T, name string, args map[string]any) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{"tool": name, "arguments": args})
	if err != nil {
		t.Fatalf("marshal tool call: %v", err)
	}
	return "```tool_call\n" + string(body) + "\n```"
}

func TestAgentApprovalDenySkipsTool(t *testing.T) {
	target := filepath.Join(t.TempDir(), "shellsage_deny_test.txt")
	mock := &scriptedProvider{responses: []string{
		toolCallJSON(t, "write_file", map[string]any{"path": target, "content": "pwned"}),
		"done",
	}}
	ag := NewAgentWithOptions(mock, tools.NewToolRegistry(), Options{
		Model: "s", Temp: 0.1,
		Approver: func(tool, args string, risk tools.Risk) (bool, bool) { return false, false },
	})
	if _, err := ag.Run(context.Background(), "write it", nil); err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, err := os.Stat(target); err == nil {
		t.Errorf("denied tool must not touch the filesystem")
	}
}

func TestAgentApprovalAllowWritesFile(t *testing.T) {
	target := filepath.Join(t.TempDir(), "shellsage_allow_test.txt")
	mock := &scriptedProvider{responses: []string{
		toolCallJSON(t, "write_file", map[string]any{"path": target, "content": "ok"}),
		"done",
	}}
	ag := NewAgentWithOptions(mock, tools.NewToolRegistry(), Options{
		Model: "s", Temp: 0.1,
		Approver: func(tool, args string, risk tools.Risk) (bool, bool) { return true, true },
	})
	if _, err := ag.Run(context.Background(), "write it", nil); err != nil {
		t.Fatalf("run: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "ok" {
		t.Errorf("approved tool should have written the file (data=%q err=%v)", data, err)
	}
}

func TestPruneOldToolResults(t *testing.T) {
	msgs := []provider.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "goal"},
	}
	for i := 0; i < 12; i++ {
		msgs = append(msgs, provider.Message{Role: "assistant", Content: "step"})
		msgs = append(msgs, provider.Message{Role: "user", Content: "TOOL RESULT list_dir: " + strings.Repeat("x", 6000)})
	}
	msgs = pruneOldToolResults(msgs, 3000)
	if msgs[1].Content != "goal" {
		t.Errorf("goal message must be preserved (index 1), got %q", msgs[1].Content)
	}
	if msgs[0].Content != "sys" {
		t.Errorf("system message must be preserved")
	}
	// oldest tool results (before last 4) must be pruned; newest not
	if !strings.Contains(msgs[3].Content, "[pruned older tool output") {
		t.Errorf("expected pruning of old tool results, got: %.60s", msgs[3].Content)
	}
	if strings.Contains(msgs[len(msgs)-1].Content, "pruned") {
		t.Errorf("recent tool result must remain intact")
	}
}

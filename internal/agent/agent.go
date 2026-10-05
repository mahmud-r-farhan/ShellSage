package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"shellsage/internal/provider"
	"shellsage/internal/tools"
)

// AgentListener receives live execution updates
type AgentListener interface {
	OnThinking(thought string)
	OnToolCall(toolName string, args string)
	OnToolResult(toolName string, result string)
	OnFinalAnswer(answer string)
}

// DefaultListener prints to stdout or can be overridden
type DefaultListener struct {
	ThinkingHandler    func(string)
	ToolCallHandler    func(string, string)
	ToolResultHandler  func(string, string)
	FinalAnswerHandler func(string)
}

func (l *DefaultListener) OnThinking(t string) {
	if l.ThinkingHandler != nil {
		l.ThinkingHandler(t)
	}
}
func (l *DefaultListener) OnToolCall(name, args string) {
	if l.ToolCallHandler != nil {
		l.ToolCallHandler(name, args)
	}
}
func (l *DefaultListener) OnToolResult(name, res string) {
	if l.ToolResultHandler != nil {
		l.ToolResultHandler(name, res)
	}
}
func (l *DefaultListener) OnFinalAnswer(a string) {
	if l.FinalAnswerHandler != nil {
		l.FinalAnswerHandler(a)
	}
}

// ApprovalFunc gates risky tool executions. It receives the tool name, raw
// arguments and the tool's risk class, and decides: allow this call? and
// optionally "allow all" for the remainder of this run.
type ApprovalFunc func(toolName, args string, risk tools.Risk) (allow bool, allowAll bool)

// Options configures an agent run.
type Options struct {
	Model         string
	Temp          float64
	MaxSteps      int           // ReAct iterations (default 12)
	MaxTime       time.Duration // wall-clock budget for Run (0 = unlimited)
	Approver      ApprovalFunc  // nil => auto-approve everything (CI / scheduled runs)
	ContextBudget int           // approx token budget before old tool results are pruned
}

// DefaultContextBudget keeps agent transcripts comfortably inside model windows.
const DefaultContextBudget = 48_000

// Agent coordinates tool execution and LLM responses
type Agent struct {
	provider provider.Provider
	tools    *tools.ToolRegistry
	opts     Options
}

// NewAgent creates a new autonomous agent (legacy constructor kept for compatibility).
func NewAgent(p provider.Provider, t *tools.ToolRegistry, model string, temp float64) *Agent {
	return NewAgentWithOptions(p, t, Options{Model: model, Temp: temp})
}

// NewAgentWithOptions creates an agent with the full option set.
func NewAgentWithOptions(p provider.Provider, t *tools.ToolRegistry, opts Options) *Agent {
	if opts.Temp <= 0 {
		opts.Temp = 0.3 // focused for tool accuracy
	}
	if opts.MaxSteps <= 0 {
		opts.MaxSteps = 12
	}
	if opts.ContextBudget <= 0 {
		opts.ContextBudget = DefaultContextBudget
	}
	return &Agent{
		provider: p,
		tools:    t,
		opts:     opts,
	}
}

// SetMaxSteps overrides the ReAct step budget.
func (a *Agent) SetMaxSteps(n int) {
	if n > 0 {
		a.opts.MaxSteps = n
	}
}

// SetApprover wires the interactive approval gate.
func (a *Agent) SetApprover(fn ApprovalFunc) { a.opts.Approver = fn }

// WithTimeout returns a derived agent whose runs are bounded by a wall-clock budget.
func (a *Agent) SetMaxTime(d time.Duration) { a.opts.MaxTime = d }

type toolCallPayload struct {
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments"`
}

// nameAlias lets models that emit {"name": ..., "parameters": ...} still work.
func (c *toolCallPayload) UnmarshalJSON(data []byte) error {
	type shadow struct {
		Tool       string          `json:"tool"`
		Name       string          `json:"name"`
		Parameters json.RawMessage `json:"parameters"`
		Arguments  json.RawMessage `json:"arguments"`
		Input      json.RawMessage `json:"input"`
	}
	var s shadow
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s.Tool != "" {
		c.Tool = s.Tool
	} else if s.Name != "" {
		c.Tool = s.Name
	} else {
		return fmt.Errorf("tool call JSON missing 'tool'/'name' key")
	}
	for _, raw := range [][]byte{s.Arguments, s.Parameters, s.Input} {
		if len(raw) > 0 {
			c.Arguments = raw
			break
		}
	}
	if c.Arguments == nil {
		c.Arguments = json.RawMessage("{}")
	}
	return nil
}

var (
	reFencedToolCall = regexp.MustCompile("(?s)```(?:tool_call|tool-call|json)\\s*\n(.*?)\n\\s*```")
	reActionBlock    = regexp.MustCompile(`(?smi)^Action:\s*([A-Za-z0-9_\-\.:]+)\s*\nAction Input:\s*(.*?)(?:\n\n|\z)`)
)

// parseToolCalls extracts every tool invocation from one assistant message.
// Supported shapes: fenced ```tool_call/```json blocks with {"tool":...} JSON,
// bare single-line JSON objects, and classic ReAct `Action:`/`Action Input:` pairs.
func parseToolCalls(content string) []toolCallPayload {
	var calls []toolCallPayload
	seen := map[string]bool{}

	add := func(payload string) {
		payload = strings.TrimSpace(payload)
		if payload == "" || seen[payload] {
			return
		}
		seen[payload] = true
		var call toolCallPayload
		if err := json.Unmarshal([]byte(payload), &call); err != nil || call.Tool == "" {
			return
		}
		calls = append(calls, call)
	}

	for _, m := range reFencedToolCall.FindAllStringSubmatch(content, -1) {
		add(m[1])
	}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "{") && strings.Contains(line, "\"tool\"") {
			add(line)
		}
	}
	for _, m := range reActionBlock.FindAllStringSubmatch(content, -1) {
		input := strings.TrimSpace(m[2])
		if !strings.HasPrefix(input, "{") {
			input = "{\"input\": " + strconv.Quote(input) + "}"
		}
		add(fmt.Sprintf("{\"tool\": %s, \"arguments\": %s}", strconv.Quote(m[1]), input))
	}
	return calls
}

// Run executes the ReAct loop until completion.
func (a *Agent) Run(ctx context.Context, goal string, listener AgentListener) (string, error) {
	if a.opts.MaxTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.opts.MaxTime)
		defer cancel()
	}

	systemPrompt := `You are ShellSage Autonomous Agent, an expert AI engineer capable of using tools to research, plan, write code, run commands, and accomplish goals.
Always plan before acting. When you need information, use search or filesystem tools.
` + a.tools.FormatToolsForPrompt() + `
Rules:
- You MAY issue several tool calls in one message; put each in its own fenced \u0060\u0060\u0060tool_call block.
- Only issue tool calls you actually need; read errors carefully before retrying a failed command.
- When a tool writes files or runs commands, verify results afterwards (list_dir / read_file / tests).
- When you have finished the task, reply with ONLY your complete final answer in plain markdown — no tool_call blocks.`

	messages := []provider.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: goal},
	}

	allowAll := false

	for step := 1; step <= a.opts.MaxSteps; step++ {
		req := &provider.ChatRequest{
			Model:       a.opts.Model,
			Messages:    messages,
			Temperature: a.opts.Temp,
			Stream:      false,
		}

		resp, err := a.provider.Chat(ctx, req)
		if err != nil {
			return "", fmt.Errorf("agent error at step %d: %w", step, err)
		}

		content := resp.Content
		calls := parseToolCalls(content)

		if len(calls) == 0 {
			if listener != nil {
				listener.OnFinalAnswer(content)
			}
			return content, nil
		}

		if thought := stripToolCalls(content); strings.TrimSpace(thought) != "" && listener != nil {
			listener.OnThinking(strings.TrimSpace(thought))
		}

		// One assistant turn per step (not per tool call): keeps history honest.
		messages = append(messages, provider.Message{Role: "assistant", Content: content})

		var results strings.Builder
		for _, call := range calls {
			argsStr := string(call.Arguments)
			if listener != nil {
				listener.OnToolCall(call.Tool, argsStr)
			}

			risk := a.tools.RiskForArgs(call.Tool, argsStr)
			if !allowAll && a.opts.Approver != nil && risk.RequiresApproval() {
				allow, all := a.opts.Approver(call.Tool, argsStr, risk)
				if all {
					allowAll = true
				}
				if !allow {
					results.WriteString(fmt.Sprintf("TOOL RESULT %s:\n[DENIED] The user rejected this operation. Do not retry it; adjust your approach or conclude.\n\n", call.Tool))
					if listener != nil {
						listener.OnToolResult(call.Tool, "[denied by user]")
					}
					continue
				}
			}

			toolRes, err := a.tools.Execute(ctx, call.Tool, argsStr)
			if err != nil {
				toolRes = fmt.Sprintf("Error executing %s: %v", call.Tool, err)
			}
			toolRes = clampString(toolRes, 18000)

			if listener != nil {
				listener.OnToolResult(call.Tool, toolRes)
			}
			results.WriteString(fmt.Sprintf("TOOL RESULT %s:\n%s\n\n", call.Tool, toolRes))
		}

		messages = append(messages, provider.Message{Role: "user", Content: strings.TrimRight(results.String(), "\n")})
		messages = pruneOldToolResults(messages, a.opts.ContextBudget)
	}

	return "Agent reached maximum step limit before finishing. Partial progress is reflected in the tool results above.", nil
}

// stripToolCalls removes fenced tool blocks so the remaining prose can be shown as "thinking".
func stripToolCalls(content string) string {
	out := reFencedToolCall.ReplaceAllString(content, "")
	out = reActionBlock.ReplaceAllString(out, "")
	return out
}

// clampString truncates overly long tool output to keep the context bounded.
func clampString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + fmt.Sprintf("\n... [output truncated: %d of %d bytes]", max, len(s))
}

// pruneOldToolResults collapses stale tool-result messages when the estimated
// context grows past the budget, keeping the goal and recent turns intact.
func pruneOldToolResults(messages []provider.Message, budgetTokens int) []provider.Message {
	est := 0
	for _, m := range messages {
		est += len(m.Content) / 4
	}
	if est <= budgetTokens {
		return messages
	}

	// Keep first 2 (system+goal) and last 4 messages untouched.
	protectTail := 4
	for i := 2; i < len(messages)-protectTail && est > budgetTokens*3/4; i++ {
		m := messages[i]
		if strings.HasPrefix(m.Content, "TOOL RESULT ") && !strings.HasPrefix(m.Content, "[pruned]") {
			head := clampString(m.Content, 420)
			note := "[pruned older tool output to save context]\n" + head
			est -= (len(m.Content) - len(note)) / 4
			messages[i] = provider.Message{Role: m.Role, Content: note}
		}
	}
	return messages
}

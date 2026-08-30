package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

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

// Agent coordinates tool execution and LLM responses
type Agent struct {
	provider provider.Provider
	tools    *tools.ToolRegistry
	model    string
	temp     float64
	maxSteps int
}

// NewAgent creates a new autonomous agent
func NewAgent(p provider.Provider, t *tools.ToolRegistry, model string, temp float64) *Agent {
	if temp <= 0 {
		temp = 0.3 // focused for tool accuracy
	}
	return &Agent{
		provider: p,
		tools:    t,
		model:    model,
		temp:     temp,
		maxSteps: 10,
	}
}

type toolCallPayload struct {
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments"`
}

// Run executes the ReAct loop until completion
func (a *Agent) Run(ctx context.Context, goal string, listener AgentListener) (string, error) {
	systemPrompt := `You are ShellSage Autonomous Agent, an expert AI engineer capable of using tools to research, plan, write code, run commands, and accomplish goals.
Always plan before acting. When you need information, use search or filesystem tools.
` + a.tools.FormatToolsForPrompt() + `
When you have finished the task and have the final response, provide your complete final answer clearly.`

	messages := []provider.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: goal},
	}

	toolCallRegex := regexp.MustCompile("(?s)```tool_call\\s*\n(.*?)\\s*```")

	for step := 1; step <= a.maxSteps; step++ {
		req := &provider.ChatRequest{
			Model:       a.model,
			Messages:    messages,
			Temperature: a.temp,
			Stream:      false,
		}

		resp, err := a.provider.Chat(ctx, req)
		if err != nil {
			return "", fmt.Errorf("agent error at step %d: %w", step, err)
		}

		content := resp.Content

		// Check if the LLM invoked a tool call
		matches := toolCallRegex.FindAllStringSubmatch(content, -1)
		if len(matches) == 0 {
			// No more tool calls, final response reached
			if listener != nil {
				listener.OnFinalAnswer(content)
			}
			return content, nil
		}

		// Tool call found
		for _, m := range matches {
			rawJSON := strings.TrimSpace(m[1])
			var call toolCallPayload
			if err := json.Unmarshal([]byte(rawJSON), &call); err != nil {
				continue
			}

			argsStr := string(call.Arguments)
			if listener != nil {
				listener.OnToolCall(call.Tool, argsStr)
			}

			// Execute tool
			toolRes, err := a.tools.Execute(ctx, call.Tool, argsStr)
			if err != nil {
				toolRes = fmt.Sprintf("Error executing %s: %v", call.Tool, err)
			}

			if listener != nil {
				listener.OnToolResult(call.Tool, toolRes)
			}

			// Feed response back to context
			messages = append(messages, provider.Message{
				Role:    "assistant",
				Content: content,
			})
			messages = append(messages, provider.Message{
				Role:    "user",
				Content: fmt.Sprintf("Tool Result for '%s':\n%s", call.Tool, toolRes),
			})
		}
	}

	return "Agent reached maximum step limit before finishing.", nil
}

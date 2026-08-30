package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ToolDef defines a tool available to the agent
type ToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Handler     func(ctx context.Context, args map[string]interface{}) (string, error) `json:"-"`
}

// ToolRegistry manages registered tools
type ToolRegistry struct {
	tools map[string]ToolDef
}

// NewToolRegistry initializes standard tools
func NewToolRegistry() *ToolRegistry {
	reg := &ToolRegistry{
		tools: make(map[string]ToolDef),
	}
	reg.registerDefaultTools()
	return reg
}

// Register adds a tool to the registry
func (r *ToolRegistry) Register(tool ToolDef) {
	r.tools[tool.Name] = tool
}

// Get finds a tool by name
func (r *ToolRegistry) Get(name string) (ToolDef, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// List returns all registered tool definitions
func (r *ToolRegistry) List() []ToolDef {
	list := make([]ToolDef, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	return list
}

// Execute parses and runs a tool invocation
func (r *ToolRegistry) Execute(ctx context.Context, name string, rawArgs string) (string, error) {
	tool, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}

	var args map[string]interface{}
	if rawArgs != "" {
		if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
			// Fallback: single string argument named "input" or "query" or "path"
			args = map[string]interface{}{"input": rawArgs}
		}
	} else {
		args = make(map[string]interface{})
	}

	return tool.Handler(ctx, args)
}

// FormatToolsForPrompt generates system instructions explaining available tools
func (r *ToolRegistry) FormatToolsForPrompt() string {
	var sb strings.Builder
	sb.WriteString("You have access to the following local developer and research tools:\n\n")

	for _, t := range r.tools {
		paramsJSON, _ := json.Marshal(t.Parameters)
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n  Parameters: `%s`\n", t.Name, t.Description, string(paramsJSON)))
	}

	sb.WriteString("\nTo invoke a tool, output a JSON block formatted exactly like this:\n")
	sb.WriteString("```tool_call\n{\"tool\": \"tool_name\", \"arguments\": {\"param\": \"value\"}}\n```\n")

	return sb.String()
}

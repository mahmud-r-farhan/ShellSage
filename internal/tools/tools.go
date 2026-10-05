package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Risk classifies what a tool can mutate; drives the agent's approval gate.
type Risk string

const (
	RiskRead  Risk = "read"  // pure inspection — always safe
	RiskWrite Risk = "write" // mutates local files
	RiskExec  Risk = "exec"  // runs shell commands
	RiskNet   Risk = "net"   // outbound network side effects (non-GET)
)

// RequiresApproval reports whether this risk class must pass the approval gate.
func (r Risk) RequiresApproval() bool {
	return r == RiskWrite || r == RiskExec || r == RiskNet
}

// ToolDef defines a tool available to the agent
type ToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Risk        Risk                   `json:"risk"`
	// DynamicRisk optionally refines the risk class per-invocation (overrides Risk).
	DynamicRisk func(args map[string]interface{}) Risk                                 `json:"-"`
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
	if tool.Risk == "" {
		tool.Risk = RiskRead
	}
	r.tools[tool.Name] = tool
}

// RiskFor reports the risk class of a tool (unknown tools default to exec-level caution).
func (r *ToolRegistry) RiskFor(name string) Risk {
	t, ok := r.tools[name]
	if !ok {
		return RiskExec
	}
	return t.Risk
}

// RiskForArgs reports the risk class for a *specific* invocation, allowing
// tools like http_request to be read-only for GET but side-effecting for POST.
func (r *ToolRegistry) RiskForArgs(name, rawArgs string) Risk {
	t, ok := r.tools[name]
	if !ok {
		return RiskExec
	}
	if t.DynamicRisk != nil {
		var args map[string]interface{}
		if rawArgs != "" {
			if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
				args = map[string]interface{}{"input": rawArgs}
			}
		}
		if args == nil {
			args = map[string]interface{}{}
		}
		if dyn := t.DynamicRisk(args); dyn != "" {
			return dyn
		}
	}
	return t.Risk
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

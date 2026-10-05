package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"shellsage/internal/agent"
	"shellsage/internal/history"
	"shellsage/internal/provider"
	"shellsage/internal/tools"
)

// ResultEnvelope is the stable --json contract for scripting (jq-friendly).
type ResultEnvelope struct {
	OK         bool                `json:"ok"`
	Provider   string              `json:"provider"`
	Model      string              `json:"model"`
	Mode       string              `json:"mode"`
	Content    string              `json:"content"`
	Usage      provider.TokenUsage `json:"usage"`
	CostUSD    float64             `json:"cost_usd"`
	DurationMS int64               `json:"duration_ms"`
	Steps      int                 `json:"steps,omitempty"`
	Error      string              `json:"error,omitempty"`
}

// CollectInput joins explicit args with piped stdin and/or an --file operand.
// Rules:
//   - explicit prompt words are used as-is (or as a title when stdin/file also provided)
//   - "-" as the prompt means "read stdin"
//   - a piped or file body is appended after the prompt as fenced context
func CollectInput(args []string, filePath string, maxBytes int64) (string, error) {
	var parts []string
	promptText := strings.TrimSpace(strings.Join(args, " "))

	var stdinData []byte
	if !IsTTY(os.Stdin) {
		limited := io.LimitReader(os.Stdin, maxBytes)
		b, err := io.ReadAll(limited)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("failed reading stdin: %w", err)
		}
		stdinData = b
	}

	var fileData []byte
	if filePath != "" {
		f, err := os.Open(filePath)
		if err != nil {
			return "", fmt.Errorf("cannot read --file: %w", err)
		}
		defer f.Close()
		b, err := io.ReadAll(io.LimitReader(f, maxBytes))
		if err != nil {
			return "", fmt.Errorf("failed reading --file: %w", err)
		}
		fileData = b
	}

	body := string(stdinData)
	if body == "" && len(fileData) > 0 {
		body = string(fileData)
		fileData = nil
	}

	if promptText == "-" {
		promptText = ""
	}

	switch {
	case promptText != "" && body != "":
		parts = append(parts, promptText, "Context:\n```\n"+strings.TrimSpace(body)+"\n```")
	case promptText != "":
		parts = append(parts, promptText)
	case body != "":
		parts = append(parts, body)
	}
	if len(fileData) > 0 {
		parts = append(parts, "File:\n```\n"+strings.TrimSpace(string(fileData))+"\n```")
	}

	out := strings.TrimSpace(strings.Join(parts, "\n\n"))
	if out == "" {
		return "", fmt.Errorf("empty input: pass a prompt, pipe via stdin, or use --file")
	}
	return out, nil
}

func (rt *Runtime) attachImages(msg *provider.Message) error {
	for _, img := range rt.Options.Images {
		uri, err := provider.LoadImageAsDataURI(img)
		if err != nil {
			return err
		}
		msg.Images = append(msg.Images, uri)
	}
	return nil
}

func costFor(model string, usage provider.TokenUsage) float64 {
	pricing, ok := history.PricingTable[model]
	if !ok {
		pricing = history.ModelPricing{InputPer1M: 1.00, OutputPer1M: 2.00}
	}
	cost := (float64(usage.PromptTokens)/1e6)*pricing.InputPer1M + (float64(usage.CompletionTokens)/1e6)*pricing.OutputPer1M
	return math.Round(cost*1e8) / 1e8
}

func (rt *Runtime) printUsage(usage provider.TokenUsage, start time.Time) {
	if rt.Options.Quiet {
		return
	}
	cost := costFor(rt.Model, usage)
	fmt.Fprintf(os.Stderr, "\n[%s · %s · %d in + %d out tokens · ~$%.4f · %v]\n",
		rt.ProvID, rt.Model, usage.PromptTokens, usage.CompletionTokens, cost, time.Since(start).Round(time.Millisecond))
}

func emitEnvelope(env ResultEnvelope) {
	data, _ := json.MarshalIndent(env, "", "  ")
	fmt.Println(string(data))
}

// RunAsk performs a one-shot chat completion.
func (rt *Runtime) RunAsk(ctx context.Context, input string) (int, error) {
	if err := rt.EnsureKey(); err != nil {
		return 1, err
	}

	sys := "You are ShellSage, a precise expert AI assistant. Answer directly and completely; use markdown when helpful. Keep formatting clean for terminal output."
	if rt.Options.SystemPrompt != "" {
		sys = rt.Options.SystemPrompt
	}

	userMsg := provider.Message{Role: "user", Content: input}
	if err := rt.attachImages(&userMsg); err != nil {
		return 2, err
	}

	req := &provider.ChatRequest{
		Model:       rt.Model,
		Messages:    []provider.Message{{Role: "system", Content: sys}, userMsg},
		Temperature: rt.Temp,
		MaxTokens:   rt.Options.MaxTokens,
	}

	start := time.Now()
	var content string
	var usage provider.TokenUsage

	if rt.Stream {
		var sb strings.Builder
		resp, err := rt.Prov.Stream(ctx, req, func(chunk string) error {
			fmt.Print(chunk)
			sb.WriteString(chunk)
			return nil
		})
		if err != nil {
			return 1, err
		}
		fmt.Println()
		content, usage = sb.String(), resp.Usage
	} else {
		resp, err := rt.Prov.Chat(ctx, req)
		if err != nil {
			return 1, err
		}
		content, usage = resp.Content, resp.Usage
		if !rt.Options.Quiet {
			fmt.Println(content)
		}
	}

	rt.printUsage(usage, start)

	if rt.Options.JSON {
		emitEnvelope(ResultEnvelope{
			OK: true, Provider: string(rt.ProvID), Model: rt.Model, Mode: "ask",
			Content: content, Usage: usage, CostUSD: costFor(rt.Model, usage),
			DurationMS: time.Since(start).Milliseconds(),
		})
	} else if !rt.Options.Quiet && rt.Stream {
		// already printed via stream
	}

	return 0, nil
}

// runOneShotPrompt runs a fixed-prompt generation (plan/debug/doc share plumbing).
func (rt *Runtime) runOneShotPrompt(ctx context.Context, mode, input string, call func(context.Context, string) (string, error)) (int, error) {
	if err := rt.EnsureKey(); err != nil {
		return 1, err
	}
	start := time.Now()
	out, err := call(ctx, input)
	if err != nil {
		if rt.Options.JSON {
			emitEnvelope(ResultEnvelope{OK: false, Mode: mode, Provider: string(rt.ProvID), Model: rt.Model, Error: err.Error(), DurationMS: time.Since(start).Milliseconds()})
		}
		return 1, err
	}
	fmt.Println(out)
	if rt.Options.JSON {
		emitEnvelope(ResultEnvelope{
			OK: true, Mode: mode, Provider: string(rt.ProvID), Model: rt.Model,
			Content: out, DurationMS: time.Since(start).Milliseconds(),
		})
	}
	return 0, nil
}

// RunPlan generates an architectural plan.
func (rt *Runtime) RunPlan(ctx context.Context, input string) (int, error) {
	return rt.runOneShotPrompt(ctx, "plan", input, rt.NewAgent().GeneratePlan)
}

// RunDebug diagnoses an error.
func (rt *Runtime) RunDebug(ctx context.Context, input string) (int, error) {
	return rt.runOneShotPrompt(ctx, "debug", input, rt.NewAgent().DiagnoseError)
}

// RunDoc generates documentation for a file path or raw code input.
func (rt *Runtime) RunDoc(ctx context.Context, input string) (int, error) {
	ag := rt.NewAgent()
	return rt.runOneShotPrompt(ctx, "doc", input, func(ctx context.Context, in string) (string, error) {
		// If the input is a single existing path, feed its contents like v3 did.
		if lines := strings.Split(strings.TrimSpace(in), "\n"); len(lines) == 1 {
			path := strings.TrimPrefix(lines[0], "@")
			if data, err := os.ReadFile(path); err == nil {
				return ag.GenerateDocumentation(ctx, path, string(data))
			}
		}
		return ag.GenerateDocumentation(ctx, "input", in)
	})
}

// RunAgent executes the autonomous ReAct loop.
func (rt *Runtime) RunAgent(ctx context.Context, goal string) (int, error) {
	if err := rt.EnsureKey(); err != nil {
		return 1, err
	}

	var logF *os.File
	if rt.Options.LogFile != "" {
		f, err := os.OpenFile(rt.Options.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return 2, fmt.Errorf("cannot open --log-file: %w", err)
		}
		defer f.Close()
		logF = f
	}

	ag := rt.NewAgent()
	start := time.Now()
	steps := 0

	listener := &agent.DefaultListener{
		ThinkingHandler: func(t string) {
			if !rt.Options.Quiet {
				color.HiBlack("  💭 %s\n", firstLine(t))
			}
			if logF != nil {
				fmt.Fprintf(logF, "[thinking] %s\n", strings.TrimSpace(t))
			}
		},
		ToolCallHandler: func(name, args string) {
			steps++
			if !rt.Options.Quiet {
				color.HiCyan("  🔧 %s %s\n", name, truncateOneLine(args, 100))
			}
			if logF != nil {
				fmt.Fprintf(logF, "[tool-call] %s %s\n", name, truncateOneLine(args, 4000))
			}
		},
		ToolResultHandler: func(name, res string) {
			if logF != nil {
				fmt.Fprintf(logF, "[tool-result] %s %s\n", name, truncateOneLine(res, 4000))
			}
		},
	}

	final, err := ag.Run(ctx, goal, listener)
	if err != nil {
		if rt.Options.JSON {
			emitEnvelope(ResultEnvelope{OK: false, Mode: "agent", Provider: string(rt.ProvID), Model: rt.Model, Error: err.Error(), Steps: steps, DurationMS: time.Since(start).Milliseconds()})
		}
		return 1, err
	}

	fmt.Println(final)

	if rt.Options.JSON {
		emitEnvelope(ResultEnvelope{
			OK: true, Mode: "agent", Provider: string(rt.ProvID), Model: rt.Model,
			Content: final, Steps: steps, DurationMS: time.Since(start).Milliseconds(),
		})
	}
	return 0, nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i > 0 {
		return s[:i]
	}
	return s
}

func truncateOneLine(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// interactiveApprover asks on the terminal; returns allow + allowAll.
func interactiveApprover() agent.ApprovalFunc {
	reader := bufio.NewReader(os.Stdin)
	return func(tool, args string, risk tools.Risk) (bool, bool) {
		return promptApproval(reader, tool, args, string(risk))
	}
}

// promptApproval renders the approval gate. Returns (allow, allowAll).
func promptApproval(reader *bufio.Reader, tool, args, risk string) (bool, bool) {
	color.Yellow("\n🔐 APPROVAL REQUIRED (%s) → tool: %s\n", risk, tool)
	preview := args
	if len(preview) > 500 {
		preview = preview[:500] + " …"
	}
	color.White("   %s\n", strings.ReplaceAll(preview, "\n", "\n   "))
	color.Cyan("   Allow? [y]es / [n]o / [a]ll for this run: ")
	line, err := reader.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	if err != nil && line == "" {
		return false, false
	}
	switch line {
	case "y", "yes":
		return true, false
	case "a", "all", "always":
		return true, true
	default:
		return false, false
	}
}

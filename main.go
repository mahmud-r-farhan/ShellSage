// ShellSage — Autonomous AI Terminal Assistant & Developer Platform.
//
// Entry point: parses subcommands (ask/agent/plan/...), otherwise starts the
// interactive REPL. One-shot & admin logic lives in internal/cli.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"shellsage/internal/agent"
	"shellsage/internal/branch"
	"shellsage/internal/cli"
	"shellsage/internal/clipboard"
	"shellsage/internal/config"
	"shellsage/internal/export"
	"shellsage/internal/history"
	"shellsage/internal/persona"
	"shellsage/internal/provider"
	"shellsage/internal/scheduler"
	"shellsage/internal/security"
	"shellsage/internal/tools"
	"shellsage/internal/tui"
)

// Version is overridden at build time via -ldflags "-X main.Version=vX.Y.Z".
var Version = "4.0.0"

// firstCommand finds the subcommand token in argv, skipping flag values
// (using the same flag tables the cli package uses for reordering).
func firstCommand(args []string) (string, int) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			name := "-" + strings.TrimLeft(strings.SplitN(a, "=", 2)[0], "-")
			if cli.GlobalValueFlags[name] && !strings.Contains(a, "=") {
				i++ // skip its value
			}
			continue
		}
		if cli.IsCommand(a) {
			return a, i
		}
		// Bare leading word that isn't a command: treat as implicit `ask` prompt.
		if i == 0 {
			return "ask", i - 1 // idx = -1 signals "no command token to strip"
		}
		return "", -1
	}
	return "", -1
}

func main() {
	os.Exit(run())
}

func run() int {
	rawArgs := os.Args[1:]

	// ── Subcommand dispatch ────────────────────────────────────────────
	if cmd, idx := firstCommand(rawArgs); cmd != "" {
		var dispatchArgs, chatArgs []string
		if idx < 0 {
			// implicit `shellsage ask "<prompt>"`: the whole argv is the ask input
			dispatchArgs = append([]string{}, rawArgs...)
			chatArgs = dispatchArgs
		} else {
			leading := append([]string{}, rawArgs[:idx]...)
			rest := append([]string{}, rawArgs[idx+1:]...)
			dispatchArgs = append(leading, rest...)
			chatArgs = dispatchArgs
		}
		if cmd == "chat" {
			// REPL continues below; drop the "chat" token from argv.
			rawArgs = chatArgs
			os.Args = append([]string{"shellsage"}, rawArgs...)
		} else if idx < 0 {
			return cli.Dispatch(cmd, dispatchArgs)
		} else {
			return cli.Dispatch(cmd, dispatchArgs)
		}
	}

	// ── Legacy flags (v1–v3 compatibility) ─────────────────────────────
	fs := flag.NewFlagSet("shellsage", flag.ExitOnError)
	versionFlag := fs.Bool("version", false, "Print ShellSage version")
	configFlag := fs.Bool("config", false, "Launch interactive configuration setup wizard")
	agentFlag := fs.String("agent", "", "Run autonomous agent with a specified goal and exit")
	planFlag := fs.String("plan", "", "Generate an implementation plan for a requirement and exit")
	debugFlag := fs.String("debug", "", "Diagnose an error message or log file and exit")
	docFlag := fs.String("doc", "", "Generate documentation for a specified file path and exit")
	auditFlag := fs.String("audit", "", "Run security and vulnerability audit on target URL or codebase and exit")
	provFlag := fs.String("provider", "", "Override active provider for this invocation")
	modelFlag := fs.String("model", "", "Override model for this invocation")
	yesFlag := fs.Bool("yes", false, "Auto-approve agent tools that mutate state")
	jsonFlag := fs.Bool("json", false, "JSON result envelope (one-shot modes)")
	quietFlag := fs.Bool("quiet", false, "Suppress decorative output")
	_ = fs.Parse(rawArgs)

	if *versionFlag {
		fmt.Printf("ShellSage AI CLI v%s\n", Version)
		return 0
	}
	cli.Version = Version

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.LoadConfig()
	if err != nil {
		color.Yellow("⚠️ Notice: %v", err)
	}

	// Legacy one-shot modes → delegate to the shared runtime.
	oneShot := func(mode, input string) int {
		o := cli.Options{
			Provider: *provFlag, Model: *modelFlag,
			Yes: *yesFlag, JSON: *jsonFlag, Quiet: *quietFlag,
		}
		o.Finalize()
		rt, err := cli.NewRuntime(o)
		if err != nil {
			color.Red("❌ %v", err)
			return 1
		}
		var code int
		switch mode {
		case "agent":
			code, err = rt.RunAgent(ctx, input)
		case "plan":
			code, err = rt.RunPlan(ctx, input)
		case "debug":
			code, err = rt.RunDebug(ctx, input)
		case "doc":
			code, err = rt.RunDoc(ctx, input)
		}
		if err != nil {
			color.Red("❌ %v\n", err)
		}
		return code
	}

	if *agentFlag != "" {
		return oneShot("agent", *agentFlag)
	}
	if *planFlag != "" {
		return oneShot("plan", *planFlag)
	}
	if *debugFlag != "" {
		return oneShot("debug", *debugFlag)
	}
	if *docFlag != "" {
		return oneShot("doc", *docFlag)
	}
	if *auditFlag != "" {
		fmt.Print(security.RunFullAudit(ctx, *auditFlag))
		return 0
	}

	if *configFlag {
		if err := tui.RunConfigWizard(cfg); err != nil {
			return 1
		}
		return 0
	}

	// ── Interactive REPL ────────────────────────────────────────────────
	app := &chatApp{cfg: cfg, ctx: ctx, reader: bufio.NewReader(os.Stdin)}
	if *provFlag != "" {
		if id, ok := provider.NormalizeID(*provFlag); ok {
			cfg.ActiveProvider = config.ProviderType(id)
		}
	}
	if *modelFlag != "" {
		p := cfg.Providers[cfg.ActiveProvider]
		p.Model = *modelFlag
		cfg.Providers[cfg.ActiveProvider] = p
	}

	if err := app.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		return 1
	}
	app.RunLoop()
	return 0
}

// chatApp owns all interactive REPL state.
type chatApp struct {
	cfg    *config.Config
	ctx    context.Context
	reader *bufio.Reader

	personaMgr    *persona.PersonaManager
	activePersona persona.Persona
	prov          provider.Provider
	model         string
	temp          float64

	toolReg *tools.ToolRegistry
	tree    *branch.ConversationTree
	stats   *history.SessionAnalytics

	queue *scheduler.TaskQueue
	sched *scheduler.Scheduler

	pendingImages []string
	allowAll      bool
}

func (a *chatApp) Init() error {
	a.personaMgr = persona.NewPersonaManager()
	if p, ok := a.personaMgr.Get("general"); ok {
		a.activePersona = p
	}

	p, err := a.cfg.NewActiveProvider()
	if err != nil {
		color.Red("❌ Error initializing provider: %v", err)
		color.Yellow("Launching setup wizard to configure provider...")
		if werr := tui.RunConfigWizard(a.cfg); werr != nil {
			return werr
		}
		p, err = a.cfg.NewActiveProvider()
		if err != nil {
			return fmt.Errorf("provider still misconfigured: %w", err)
		}
	}
	a.prov = p

	pCfg, _ := a.cfg.GetActiveProviderConfig()
	a.model = pCfg.Model
	a.temp = a.cfg.DefaultTemp
	a.toolReg = tools.NewToolRegistry()
	a.tree = branch.NewConversationTree(a.activePersona.ID)
	a.stats = history.NewSessionAnalytics()
	a.queue = scheduler.NewTaskQueue(a.newAgent())
	a.sched = scheduler.NewScheduler(a.newAgent())

	// Graceful termination: save, then exit.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		if a.cfg.AutoSaveHistory && len(a.tree.Nodes) > 0 {
			_, _ = history.SaveTree(a.tree, "")
		}
		color.Green("\n✨ Goodbye! Session ended gracefully.\n")
		os.Exit(0)
	}()

	a.sched.Start(a.ctx, func(j *scheduler.ScheduledJob) {
		color.HiMagenta("\n🔔 [SCHEDULER] Task '%s' completed at %s!\n", j.Goal, time.Now().Format("15:04:05"))
		if j.Error != "" {
			color.Red("   Error: %s\n", j.Error)
		} else {
			color.Green("   Result: %s\n", truncateString(j.Result, 120))
		}
		color.Cyan("👤 You: ")
	})

	tui.PrintBanner(string(a.cfg.ActiveProvider), a.model, a.activePersona.Name)
	return nil
}

func (a *chatApp) newAgent() *agent.Agent {
	opts := agent.Options{Model: a.model, Temp: a.temp}
	if a.cfg.AgentMaxSteps > 0 {
		opts.MaxSteps = a.cfg.AgentMaxSteps
	}
	ag := agent.NewAgentWithOptions(a.prov, a.toolReg, opts)
	ag.SetApprover(a.approver())
	return ag
}

func (a *chatApp) approver() agent.ApprovalFunc {
	mode := strings.ToLower(a.cfg.ApproveMode)
	if a.allowAll || mode == "yolo" {
		return nil
	}
	if mode == "read-only" {
		return func(tool, args string, risk tools.Risk) (bool, bool) { return false, false }
	}
	return func(tool, args string, risk tools.Risk) (bool, bool) {
		color.Yellow("\n🔐 APPROVAL REQUIRED (%s) → tool: %s\n", risk, tool)
		preview := truncateString(args, 500)
		color.White("   %s\n", preview)
		color.Cyan("   Allow? [y]es / [n]o / [a]ll for this session: ")
		line, _ := a.reader.ReadString('\n')
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "y", "yes":
			return true, false
		case "a", "all", "always":
			a.allowAll = true
			return true, true
		default:
			return false, false
		}
	}
}

func (a *chatApp) rewire() {
	// Refresh the agents used by the queue & scheduler after
	// provider/model/persona changes (scheduler keeps its live timer).
	a.queue.SetAgent(a.newAgent())
	a.sched.SetAgent(a.newAgent())
}

func (a *chatApp) RunLoop() {
	for {
		color.Cyan("\n👤 You: ")
		userInput, err := a.reader.ReadString('\n')
		if err != nil {
			break
		}
		userInput = strings.TrimSpace(userInput)
		if userInput == "" {
			continue
		}

		if strings.HasPrefix(userInput, "/") {
			if quit := a.handleSlash(strings.Fields(userInput)); quit {
				return
			}
			continue
		}

		userMsg := provider.Message{Role: "user", Content: userInput, Images: a.pendingImages}
		a.tree.AddMessageWithParts(userMsg, a.model, provider.TokenUsage{})
		a.pendingImages = nil
		a.generate()
	}
}

// handleSlash dispatches a slash command; returns true if the REPL should exit.
func (a *chatApp) handleSlash(parts []string) bool {
	cmd := strings.ToLower(parts[0])
	switch cmd {
	case "/exit", "/quit":
		a.saveIfNeeded()
		fmt.Print(a.stats.FormatSummary(a.activePersona.Name, a.model, a.tree))
		color.Green("✨ Goodbye! Thanks for using ShellSage v%s.\n\n", Version)
		a.sched.Stop()
		return true

	case "/help":
		tui.PrintHelp()
	case "/config":
		if err := tui.RunConfigWizard(a.cfg); err == nil {
			if p, err := a.cfg.NewActiveProvider(); err == nil {
				a.prov = p
				pCfg, _ := a.cfg.GetActiveProviderConfig()
				a.model = pCfg.Model
				a.rewire()
				color.Green("✅ Provider & Model updated to %s (%s)\n", a.cfg.ActiveProvider, a.model)
			}
		}
	case "/provider":
		newProv := tui.SelectProviderMenu(a.cfg.ActiveProvider)
		if newProv != a.cfg.ActiveProvider {
			a.cfg.ActiveProvider = newProv
			if p, err := a.cfg.NewActiveProvider(); err == nil {
				a.prov = p
				pCfg, _ := a.cfg.GetActiveProviderConfig()
				a.model = pCfg.Model
				a.rewire()
				color.Green("✅ Switched provider to %s (Model: %s)\n", newProv, a.model)
			} else {
				color.Red("Error switching provider: %v\n", err)
			}
		}
	case "/model":
		newModel := tui.SelectModelMenu(a.model, a.prov.ListAvailableModels())
		if newModel != "" && newModel != a.model {
			a.model = newModel
			pCfg, _ := a.cfg.GetActiveProviderConfig()
			pCfg.Model = newModel
			a.cfg.Providers[a.cfg.ActiveProvider] = pCfg
			a.rewire()
			color.Green("✅ Model switched to: %s\n", a.model)
		}
	case "/models":
		fmt.Printf("Available models for %s:\n", a.cfg.ActiveProvider)
		for _, m := range a.prov.ListAvailableModels() {
			marker := "  "
			if m == a.model {
				marker = "→ "
			}
			fmt.Printf(" %s%s\n", marker, m)
		}
		color.HiBlack(" Use /model to switch, or `shellsage models list --remote` for the live catalog.\n")
	case "/persona":
		if len(parts) > 1 && strings.ToLower(parts[1]) == "create" {
			a.activePersona = tui.CreatePersonaWizard(a.personaMgr)
		} else if len(parts) > 1 {
			if p, ok := a.personaMgr.Get(parts[1]); ok {
				a.activePersona = p
			} else {
				color.Yellow("Unknown persona %q — use /persona to pick one\n", parts[1])
			}
		} else {
			a.activePersona = tui.SelectPersonaMenu(a.personaMgr, a.activePersona.ID)
		}
		a.tree.PersonaID = a.activePersona.ID
		color.Green("✅ Active Persona set to: %s\n", a.activePersona.Name)
	case "/temp":
		a.temp = tui.AdjustTemperatureMenu(a.temp)
		a.rewire()
		color.Green("✅ Temperature adjusted to: %.2f\n", a.temp)
	case "/approve":
		if len(parts) > 1 {
			switch strings.ToLower(parts[1]) {
			case "ask":
				a.cfg.ApproveMode = "ask"
				a.allowAll = false
			case "yolo", "auto":
				a.cfg.ApproveMode = "yolo"
			case "read-only", "readonly":
				a.cfg.ApproveMode = "read-only"
			default:
				color.Yellow("Usage: /approve <ask|yolo|read-only>\n")
				return false
			}
			a.rewire()
			color.Green("✅ Agent approval mode: %s\n", a.cfg.ApproveMode)
		} else {
			color.Cyan("Approval mode: %s (yolo = auto-allow mutating tools, read-only = refuse)\n", a.cfg.ApproveMode)
		}
	case "/clear":
		a.saveIfNeeded()
		a.tree = branch.NewConversationTree(a.activePersona.ID)
		color.Green("✅ Conversation history cleared! Started fresh tree.\n")
	case "/compress":
		a.compressContext()
	case "/image":
		if len(parts) < 2 {
			color.Yellow("Usage: /image <file.png|https://...>  (attaches to your NEXT message; %d pending)\n", len(a.pendingImages))
		} else {
			uri, err := provider.LoadImageAsDataURI(parts[1])
			if err != nil {
				color.Red("❌ %v\n", err)
			} else {
				a.pendingImages = append(a.pendingImages, uri)
				color.Green("✅ Image attached (%d pending) — send your message now.\n", len(a.pendingImages))
			}
		}
	case "/copy":
		a.handleCopy(parts)
	case "/export":
		a.handleExport(parts)
	case "/retry", "/alt":
		if userNode, ok := a.tree.RewindForAlternative(); ok {
			color.Yellow("🔄 Generating alternative response for: %q...\n", truncateString(userNode.Content, 40))
			a.generate()
		} else {
			color.Yellow("⚠️ Cannot rewind: no preceding user turn found to regenerate.\n")
		}
	case "/branch":
		tui.SelectBranchMenu(a.tree)
	case "/tree":
		fmt.Print(a.tree.RenderTreeAscii())
	case "/stats", "/analytics":
		fmt.Print(a.stats.FormatSummary(a.activePersona.Name, a.model, a.tree))
	case "/agent":
		if len(parts) > 1 {
			a.runAgent(strings.Join(parts[1:], " "))
		} else {
			color.Yellow("Usage: /agent <goal description>\n")
		}
	case "/plan":
		if len(parts) > 1 {
			a.runGen("plan", strings.Join(parts[1:], " "))
		} else {
			color.Yellow("Usage: /plan <feature or system requirement>\n")
		}
	case "/debug":
		if len(parts) > 1 {
			a.runGen("debug", strings.Join(parts[1:], " "))
		} else {
			color.Yellow("Usage: /debug <error message or stack trace>\n")
		}
	case "/doc":
		if len(parts) > 1 {
			a.runGen("doc", parts[1])
		} else {
			color.Yellow("Usage: /doc <file path>\n")
		}
	case "/tools":
		a.printTools()
	case "/queue":
		a.handleQueue(parts)
	case "/schedule":
		a.handleSchedule(parts)
	case "/sec":
		cli.SecReport(a.ctx, parts[1:])
	case "/audit":
		target := "."
		if len(parts) > 1 {
			target = parts[1]
		}
		fmt.Print(security.RunFullAudit(a.ctx, target))
	case "/save":
		fname := ""
		if len(parts) > 1 {
			fname = parts[1]
		}
		savedPath, err := history.SaveTree(a.tree, fname)
		if err != nil {
			color.Red("Error saving: %v\n", err)
		} else {
			color.Green("✅ Conversation saved to %s\n", savedPath)
		}
	case "/load":
		a.loadSession(parts)
	case "/resume":
		name := ""
		if len(parts) > 1 {
			name = parts[1]
		} else if latest, ok := history.LatestConversation(); ok {
			name = latest
		}
		if name == "" {
			color.Yellow("Nothing to resume (no saved conversations).\n")
		} else {
			a.loadSession([]string{"/load", name})
		}
	case "/rename":
		if len(parts) > 1 {
			a.tree.Title = strings.Join(parts[1:], " ")
			color.Green("✅ Session titled: %s\n", a.tree.Title)
		} else {
			color.Yellow("Usage: /rename <new session title>\n")
		}
	case "/delete":
		if len(parts) > 1 {
			if path, err := history.DeleteTree(parts[1]); err != nil {
				color.Red("❌ %v\n", err)
			} else {
				color.Green("🗑 Deleted %s\n", path)
			}
		} else {
			color.Yellow("Usage: /delete <saved-session-file> (see /list)\n")
		}
	case "/list":
		showSavedConversations()
	case "/history":
		limit := 10
		if len(parts) > 1 {
			if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 {
				limit = n
			}
		}
		showActiveHistory(a.tree, limit)
	case "/search":
		if len(parts) > 1 {
			searchActiveTree(a.tree, strings.Join(parts[1:], " "))
		} else {
			color.Yellow("Usage: /search <term>\n")
		}
	default:
		color.Yellow("❌ Unknown command: %s (Type /help for command list)\n", parts[0])
	}
	return false
}

// ── Generation with context budget management ────────────────────────────

func (a *chatApp) buildMessages() []provider.Message {
	messages := make([]provider.Message, 0, len(a.tree.GetActivePath())+1)
	messages = append(messages, provider.Message{Role: "system", Content: a.activePersona.Prompt})
	messages = append(messages, a.tree.GetActiveMessages()...)

	// Context budget: trim oldest turns (keep system + last 6) when over.
	budget := a.cfg.MaxContextTokens
	if budget > 0 {
		est := 0
		for _, m := range messages {
			est += len(m.Content)/4 + len(m.Images)*850
		}
		if est > budget && len(messages) > 8 {
			drop := 1
			for est > budget*3/4 && len(messages)-drop > 7 {
				est -= len(messages[drop].Content) / 4
				drop++
			}
			if drop > 1 {
				trimmed := make([]provider.Message, 0, len(messages)-drop+2)
				trimmed = append(trimmed, messages[0])
				trimmed = append(trimmed, provider.Message{Role: "system", Content: fmt.Sprintf("[ShellSage trimmed %d older message(s) to fit the %d-token context budget — use /compress to fold them into a smart summary instead.]", drop-1, budget)})
				trimmed = append(trimmed, messages[drop:]...)
				messages = trimmed
			}
		}
	}
	return messages
}

func (a *chatApp) generate() {
	req := &provider.ChatRequest{
		Model:       a.model,
		Messages:    a.buildMessages(),
		Temperature: a.temp,
		Stream:      a.cfg.StreamResponses,
	}

	color.Yellow("\n🤖 Assistant: ")

	if req.Stream {
		var sb strings.Builder
		resp, err := a.prov.Stream(a.ctx, req, func(chunk string) error {
			fmt.Print(chunk)
			sb.WriteString(chunk)
			return nil
		})
		if err != nil {
			color.Red("\n❌ Stream error: %v\n", err)
			return
		}
		fmt.Println()
		a.recordAssistant(resp)
	} else {
		resp, err := a.prov.Chat(a.ctx, req)
		if err != nil {
			color.Red("\n❌ API error: %v\n", err)
			return
		}
		fmt.Println(resp.Content)
		a.recordAssistant(resp)
	}
}

func (a *chatApp) recordAssistant(resp *provider.ChatResponse) {
	a.tree.AddMessage("assistant", resp.Content, a.model, resp.Usage)
	a.stats.RecordTurn(a.model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
	color.HiBlack("[Tokens: +%d in, +%d out | Total: %d]\n",
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens, a.stats.TotalTokens)
}

// compressContext folds all but the most recent 4 turns into an LLM summary.
func (a *chatApp) compressContext() {
	path := a.tree.GetActivePath()
	if len(path) < 6 {
		color.Yellow("Nothing worth compressing yet (conversation shorter than 6 turns).\n")
		return
	}
	keepFrom := len(path) - 4
	var sb strings.Builder
	for _, n := range path[:keepFrom] {
		sb.WriteString(fmt.Sprintf("[%s]: %s\n\n", strings.ToUpper(n.Role), truncateString(n.Content, 3000)))
	}

	color.Yellow("\n🗜  Compressing %d older turn(s) into a context summary...\n", keepFrom)
	req := &provider.ChatRequest{
		Model: a.model,
		Messages: []provider.Message{
			{Role: "system", Content: "You compress conversation history for an AI coding assistant. Write a dense, information-preserving summary. You MUST keep: explicit decisions, file paths, function names, code snippets (shortened OK), error messages, open TODOs and the user's stated intent. Bullet points only, no preamble."},
			{Role: "user", Content: sb.String()},
		},
		Temperature: 0.2,
	}
	resp, err := a.prov.Chat(a.ctx, req)
	if err != nil {
		color.Red("❌ Compression failed (provider error): %v\n", err)
		return
	}

	rebuilt := branch.NewConversationTree(a.tree.PersonaID)
	rebuilt.Title = a.tree.Title
	rebuilt.AddMessage("assistant", "[CONTEXT SUMMARY of earlier turns]\n"+resp.Content, a.model, resp.Usage)
	for _, n := range path[keepFrom:] {
		rebuilt.AddMessageWithParts(provider.Message{Role: n.Role, Content: n.Content}, a.model, n.Usage)
	}
	// Preserve branch history by re-anchoring: the summary becomes the only path.
	a.tree = rebuilt
	a.stats.RecordTurn(a.model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	freed := len(path) - 4 - 1
	color.Green("✅ Compressed %d turns into 1 summary node (~%d tokens used). The full original transcript stays available via /save before compression or export.\n", freed, resp.Usage.PromptTokens)
}

// ── Agent / one-shot runners inside the REPL ─────────────────────────────

func (a *chatApp) runAgent(goal string) {
	color.Cyan("\n🤖 ━━ Autonomous Agent Initiated ━━\n")
	color.White("Goal: %s\n\n", goal)

	ag := a.newAgent()
	listener := &agent.DefaultListener{
		ThinkingHandler: func(t string) {
			color.HiBlack("  💭 %s\n", truncateString(t, 110))
		},
		ToolCallHandler: func(name, args string) {
			color.HiCyan("  🔧 Executing tool [%s] with args: %s\n", name, truncateString(args, 100))
		},
		ToolResultHandler: func(name, res string) {
			color.HiBlack("     ↳ Result (%d chars): %s\n", len(res), truncateString(res, 80))
		},
		FinalAnswerHandler: func(ans string) {
			color.Green("\n🎯 Final Agent Response:\n\n")
			fmt.Println(ans)
		},
	}

	res, err := ag.Run(a.ctx, goal, listener)
	if err != nil {
		color.Red("\n❌ Agent encountered an error: %v\n", err)
		return
	}
	a.tree.AddMessage("user", "/agent "+goal, a.model, provider.TokenUsage{})
	a.tree.AddMessage("assistant", res, a.model, provider.TokenUsage{})
}

func (a *chatApp) runGen(mode, input string) {
	ag := a.newAgent()
	var out string
	var err error
	switch mode {
	case "plan":
		color.Yellow("\n📐 Generating Architectural Plan...\n")
		out, err = ag.GeneratePlan(a.ctx, input)
	case "debug":
		color.Yellow("\n🔍 Analyzing Error and Root Cause...\n")
		out, err = ag.DiagnoseError(a.ctx, input)
	case "doc":
		color.Yellow("\n📖 Generating Documentation for %s...\n", input)
		content := ""
		if data, rerr := os.ReadFile(input); rerr == nil {
			content = string(data)
		} else {
			color.Red("Error reading file '%s': %v (using path as prompt)\n", input, rerr)
		}
		out, err = ag.GenerateDocumentation(a.ctx, input, content)
	}
	if err != nil {
		color.Red("Error: %v\n", err)
		return
	}
	fmt.Println(out)
	a.tree.AddMessage("user", "/"+mode+" "+truncateString(input, 120), a.model, provider.TokenUsage{})
	a.tree.AddMessage("assistant", out, a.model, provider.TokenUsage{})
}

func (a *chatApp) printTools() {
	color.Cyan("\n🧰 ━━ Agent Tool Inventory ━━\n")
	for _, t := range a.toolReg.List() {
		badge := color.GreenString("read")
		switch t.Risk {
		case tools.RiskWrite:
			badge = color.YellowString("write")
		case tools.RiskExec:
			badge = color.RedString("exec ")
		case tools.RiskNet:
			badge = color.MagentaString("net  ")
		}
		color.White("  [%s] %-14s %s\n", badge, t.Name, color.HiBlackString(firstLineOf(t.Description)))
	}
	fmt.Println()
	color.HiBlack("  Approval mode: %s — risky tools prompt for confirmation (or run `shellsage agent --yes ...`).\n", a.cfg.ApproveMode)
}

func firstLineOf(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func (a *chatApp) saveIfNeeded() {
	if a.cfg.AutoSaveHistory && len(a.tree.Nodes) > 0 {
		savedPath, _ := history.SaveTree(a.tree, "")
		if savedPath != "" {
			color.HiBlack("💾 Session auto-saved to %s\n", savedPath)
		}
	}
}

func (a *chatApp) loadSession(parts []string) {
	load := func(name string) {
		loaded, err := history.LoadTree(name)
		if err != nil {
			color.Red("Error loading: %v\n", err)
			return
		}
		a.tree = loaded
		if p, ok := a.personaMgr.Get(treePersona(loaded, a.personaMgr)); ok {
			a.activePersona = p
		}
		color.Green("✅ Loaded conversation '%s' (%d turns)\n", loaded.Title, len(loaded.GetActivePath()))
	}

	if len(parts) > 1 {
		load(parts[1])
		return
	}
	showSavedConversations()
	color.Cyan("Enter filename to load: ")
	input, _ := a.reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		load(input)
	}
}

func treePersona(t *branch.ConversationTree, mgr *persona.PersonaManager) string {
	if t.PersonaID != "" {
		return t.PersonaID
	}
	return "general"
}

// ── Copy / export / queue / schedule handlers ────────────────────────────

func (a *chatApp) handleCopy(parts []string) {
	lastNode := a.tree.GetLastMessage()
	if lastNode == nil {
		color.Yellow("⚠️ Conversation is empty, nothing to copy.\n")
		return
	}

	sub := ""
	if len(parts) > 1 {
		sub = strings.ToLower(parts[1])
	}

	switch sub {
	case "code":
		if lastNode.Role != "assistant" {
			color.Yellow("⚠️ Last message is not an assistant response.\n")
			return
		}
		code, err := clipboard.CopyCodeOnly(lastNode.Content)
		if err != nil {
			color.Red("❌ Clipboard error: %v\n", err)
		} else {
			color.Green("✅ Copied code block(s) to system clipboard!\n")
			color.HiBlack("--- Copied Snippet ---\n%s\n----------------------\n", truncateString(code, 200))
		}
	case "all":
		md := export.ExportToMarkdown(a.tree)
		if err := clipboard.CopyText(md); err != nil {
			color.Red("❌ Clipboard error: %v\n", err)
		} else {
			color.Green("✅ Copied entire conversation transcript to system clipboard!\n")
		}
	default:
		if lastNode.Role != "assistant" {
			color.Yellow("⚠️ Last message is not an assistant response.\n")
			return
		}
		if err := clipboard.CopyText(lastNode.Content); err != nil {
			color.Red("❌ Clipboard error: %v\n", err)
		} else {
			color.Green("✅ Last assistant response copied to clipboard!\n")
		}
	}
}

func (a *chatApp) handleExport(parts []string) {
	if len(parts) < 2 {
		color.Yellow("Usage: /export <md|pdf|html|json> [optional_filename]\n")
		return
	}

	fmtType := export.ExportFormat(strings.ToLower(parts[1]))
	fname := ""
	if len(parts) > 2 {
		fname = parts[2]
	}

	outPath, err := export.ExportConversation(a.tree, fmtType, fname)
	if err != nil {
		color.Red("❌ Export failed: %v\n", err)
	} else {
		color.Green("✅ Conversation successfully exported to %s\n", outPath)
	}
}

func (a *chatApp) handleQueue(parts []string) {
	if len(parts) < 2 {
		color.Yellow("Usage: /queue <add|list|run> [arguments]\n")
		return
	}

	q := a.queue
	switch strings.ToLower(parts[1]) {
	case "add":
		if len(parts) < 3 {
			color.Yellow("Usage: /queue add <task description>\n")
			return
		}
		desc := strings.Join(parts[2:], " ")
		task := q.Add(desc)
		color.Green("✅ Added task [%s]: %s\n", task.ID, task.Description)

	case "list":
		tasks := q.List()
		if len(tasks) == 0 {
			color.Yellow("Task queue is empty.\n")
			return
		}
		color.Cyan("\n📋 ━━ Task Queue (%d tasks) ━━\n", len(tasks))
		for i, t := range tasks {
			statusColor := color.YellowString
			if t.Status == scheduler.StatusCompleted {
				statusColor = color.GreenString
			} else if t.Status == scheduler.StatusFailed {
				statusColor = color.RedString
			}
			color.White("  [%d] %-10s %-12s: %s\n", i+1, t.ID, statusColor(string(t.Status)), t.Description)
		}
		fmt.Println()

	case "run":
		color.Yellow("⚙️ Processing next queued task...\n")
		listener := &agent.DefaultListener{
			ToolCallHandler: func(name, args string) {
				color.HiCyan("  🔧 [Tool: %s] args: %s", name, truncateString(args, 80))
			},
		}
		task, err := q.ExecuteNext(a.ctx, listener)
		if err != nil {
			color.Red("❌ Task failed: %v\n", err)
		} else {
			color.Green("\n✅ Task [%s] Completed!\nResult:\n%s\n", task.ID, task.Result)
		}
	}
}

func (a *chatApp) handleSchedule(parts []string) {
	s := a.sched
	if len(parts) < 2 {
		color.Yellow("Usage: /schedule <at|in|daily|list|cancel> [arguments]\nExample: /schedule at 15:30 Check status\n")
		return
	}

	switch strings.ToLower(parts[1]) {
	case "at":
		if len(parts) < 4 {
			color.Yellow("Usage: /schedule at <HH:MM> <goal>\nExample: /schedule at 16:00 Run integration tests\n")
			return
		}
		timeParts := strings.Split(parts[2], ":")
		if len(timeParts) != 2 {
			color.Red("Invalid time format. Use HH:MM (e.g. 14:30)\n")
			return
		}
		hour, _ := strconv.Atoi(timeParts[0])
		min, _ := strconv.Atoi(timeParts[1])
		goal := strings.Join(parts[3:], " ")

		job, err := s.ScheduleAt(hour, min, goal)
		if err != nil {
			color.Red("Error scheduling: %v\n", err)
		} else {
			color.Green("✅ Scheduled task [%s] for %s (survives restart if ShellSage runs again)!\n", job.ID, job.TargetTime.Format("Jan 02 15:04:05"))
		}

	case "in":
		if len(parts) < 4 {
			color.Yellow("Usage: /schedule in <duration> <goal>\nExample: /schedule in 30m Check server health\n")
			return
		}
		dur, err := time.ParseDuration(parts[2])
		if err != nil {
			color.Red("Invalid duration format (e.g. 10m, 1h, 45s): %v\n", err)
			return
		}
		goal := strings.Join(parts[3:], " ")
		job, err := s.ScheduleIn(dur, goal)
		if err != nil {
			color.Red("Error scheduling: %v\n", err)
		} else {
			color.Green("✅ Scheduled task [%s] to run in %v (at %s)!\n", job.ID, dur, job.TargetTime.Format("15:04:05"))
		}

	case "daily":
		if len(parts) < 4 {
			color.Yellow("Usage: /schedule daily <HH:MM> <goal>\nExample: /schedule daily 09:00 Summarize overnight CI failures\n")
			return
		}
		timeParts := strings.Split(parts[2], ":")
		if len(timeParts) != 2 {
			color.Red("Invalid time format. Use HH:MM\n")
			return
		}
		hour, _ := strconv.Atoi(timeParts[0])
		min, _ := strconv.Atoi(timeParts[1])
		goal := strings.Join(parts[3:], " ")
		job, err := s.ScheduleDaily(hour, min, goal)
		if err != nil {
			color.Red("Error scheduling: %v\n", err)
		} else {
			color.Green("✅ Daily task [%s] armed for %s each day!\n", job.ID, parts[2])
		}

	case "cancel":
		if len(parts) < 3 {
			color.Yellow("Usage: /schedule cancel <job_id>\n")
			return
		}
		if s.Cancel(parts[2]) {
			color.Green("✅ Cancelled job %s\n", parts[2])
		} else {
			color.Yellow("No pending job with id %s\n", parts[2])
		}

	case "list":
		jobs := s.ListJobs()
		if len(jobs) == 0 {
			color.Yellow("No scheduled jobs found.\n")
			return
		}
		color.Cyan("\n⏰ ━━ Scheduled Jobs ━━\n")
		for _, j := range jobs {
			execStatus := color.YellowString("PENDING")
			if j.Executed {
				if j.Error != "" {
					execStatus = color.RedString("FAILED")
				} else {
					execStatus = color.GreenString("EXECUTED")
				}
			} else if j.Recurring {
				execStatus = color.CyanString("DAILY")
			}
			color.White("  [%s] Target: %s | Status: %s\n    Goal: %s\n", j.ID, j.TargetTime.Format("15:04:05"), execStatus, j.Goal)
		}
		fmt.Println()
	}
}

// ── Free helpers (kept compatible with v3 behavior) ──────────────────────

func showSavedConversations() {
	list := history.ListSavedConversations()
	if len(list) == 0 {
		color.Yellow("No saved conversations found (%s).\n", history.GetConversationsDir())
		return
	}

	color.Cyan("\n💾 ━━ Saved Conversations ━━\n")
	for i, c := range list {
		color.White("  [%d] %-30s %s (%d turns, %d tokens)\n", i+1, c.Filename, color.HiBlackString("["+c.Title+"]"), c.MessageCount, c.TotalTokens)
	}
	fmt.Println()
}

func showActiveHistory(tree *branch.ConversationTree, limit int) {
	path := tree.GetActivePath()
	if len(path) == 0 {
		color.Yellow("No messages in active conversation path.\n")
		return
	}

	color.Cyan("\n📜 ━━ Last %d Messages ━━\n", limit)
	start := len(path) - limit
	if start < 0 {
		start = 0
	}

	for i := start; i < len(path); i++ {
		node := path[i]
		if node.Role == "user" {
			color.Cyan("\n👤 You (%s):\n", node.Timestamp.Format("15:04:05"))
		} else {
			color.Yellow("\n🤖 Assistant (%s):\n", node.Timestamp.Format("15:04:05"))
		}
		color.White(node.Content + "\n")
	}
}

func searchActiveTree(tree *branch.ConversationTree, query string) {
	path := tree.GetActivePath()
	queryLower := strings.ToLower(query)
	found := 0

	color.Cyan("\n🔍 Search Results for %q:\n", query)
	for i, node := range path {
		if strings.Contains(strings.ToLower(node.Content), queryLower) {
			found++
			color.Yellow("\n[Message %d - %s]:\n", i+1, node.Role)
			color.White(node.Content + "\n")
		}
	}

	if found == 0 {
		color.Yellow("No matching messages found.\n")
	} else {
		color.Green("\n✅ Found %d matching message(s)\n", found)
	}
}

func truncateString(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

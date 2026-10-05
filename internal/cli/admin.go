package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/fatih/color"
	"shellsage/internal/config"
	"shellsage/internal/history"
	"shellsage/internal/provider"
	"shellsage/internal/tui"
)

// Commands lists every supported subcommand (used by dispatcher & completion).
var Commands = []string{
	"chat", "ask", "agent", "plan", "debug", "doc",
	"audit", "sec",
	"config", "provider", "models", "sessions",
	"doctor", "completion", "version", "help",
}

// IsCommand reports whether arg names a subcommand.
func IsCommand(arg string) bool {
	for _, c := range Commands {
		if c == arg {
			return true
		}
	}
	return false
}

func newFlagSet(name string, o *Options) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	RegisterGlobalFlags(fs, o)
	return fs
}

// Dispatch routes a subcommand invocation; returns process exit code.
func Dispatch(cmd string, args []string) int {
	switch cmd {
	case "chat":
		return runChat(args)
	case "ask":
		return runAsk(args)
	case "agent":
		return runAgentCmd(args)
	case "plan":
		return runGenCmd(args, "plan")
	case "debug":
		return runGenCmd(args, "debug")
	case "doc":
		return runGenCmd(args, "doc")
	case "audit":
		return runAudit(args)
	case "sec":
		return runSecCmd(args)
	case "config":
		return runConfigCmd(args)
	case "provider":
		return runProviderCmd(args)
	case "models":
		return runModelsCmd(args)
	case "sessions":
		return runSessionsCmd(args)
	case "doctor":
		return runDoctor(args)
	case "completion":
		return runCompletion(args)
	case "version":
		fmt.Printf("ShellSage %s\n", Version)
		return 0
	case "help":
		PrintUsage()
		return 0
	}
	color.Red("Unknown command: %s\n", cmd)
	PrintUsage()
	return 2
}

// Version is injected at build time (ldflags) with a sane default.
var Version = "4.0.0"

func PrintUsage() {
	fmt.Print(`ShellSage ` + Version + ` — Autonomous AI Terminal Assistant (20 providers, scripting-ready)

Usage:
  shellsage [subcommand] [global flags] [arguments]

Chat & one-shot modes:
  chat                              Interactive REPL with the full slash-command set (default)
  ask "<prompt>"                     One-shot chat; supports pipes & files (see below)
  agent "<goal>"                     Autonomous ReAct agent with local tools
  plan "<requirement>"               Architecture & implementation plan
  debug "<error>"                    Root-cause diagnosis + fix (accepts piped logs)
  doc "<path|code>"                  Generate documentation for a file or pasted code
  audit [path|url]                   Full security posture audit
  sec <headers|ssl|ports|sast|audit> <target>

Config & capability management:
  config wizard | show | path | set <provider>.<field>=<value>
  provider list | show <id> | use <id>
  models list [--remote] | use <model-id>
  sessions list | delete <file> | rename <old> <new>
  doctor                             End-to-end connectivity & config diagnostics

Shell integration:
  completion <bash|zsh>              Emit a shell completion script
  version | help

Global flags (all subcommands):
  --provider <id>   --model <id>   --temp <0..2>   --max-tokens <n>
  --stream <auto|on|off>   --json   --quiet   --yes   --no-color
  --timeout <dur>   --config-file <path>   --system "<prompt>"
  --image <file|url> (repeatable)
  agent only: --steps <n>  --max-time <dur>  --log-file <path>
  ask/plan/debug/doc: read prompt from stdin or --file by piping ('-' prompt)

Examples:
  cat stack.log | shellsage debug - --provider groq
  shellsage ask "summarize this" --json | jq -r .content
  go test ./... 2>&1 | shellsage ask "why is this failing?"
  shellsage agent --yes "fix the failing unit tests in ./internal"
  git diff | shellsage sec sast . && shellsage provider use anthropic
`)
}

func runChat(args []string) int {
	// The REPL lives in main; this signals main to start chat with parsed opts.
	// (handled specially by Dispatch's caller — see main.go)
	color.Yellow("chat should be dispatched by main")
	return 0
}

func parseInputFlags(fs *flag.FlagSet) (string, string) {
	var file string
	fs.StringVar(&file, "file", "", "read input body from a file")
	return file, ""
}

func (rt *Runtime) fatalErr(err error) int {
	if rt != nil && rt.Options.JSON {
		emitEnvelope(ResultEnvelope{OK: false, Error: err.Error()})
	}
	color.Red("❌ %v\n", err)
	return 1
}

func runAsk(args []string) int {
	var o Options
	fs := newFlagSet("ask", &o)
	file, _ := parseInputFlags(fs)
	_ = fs.Parse(ReorderArgs(args))
	o.Finalize()

	rt, err := NewRuntime(o)
	if err != nil {
		color.Red("❌ %v\n", err)
		return 1
	}
	input, err := CollectInput(fs.Args(), file, 512*1024)
	if err != nil {
		return rt.fatalErr(err)
	}
	code, err := rt.RunAsk(context.Background(), input)
	if err != nil {
		return rt.fatalErr(err)
	}
	return code
}

func runAgentCmd(args []string) int {
	var o Options
	fs := newFlagSet("agent", &o)
	file, _ := parseInputFlags(fs)
	_ = fs.Parse(ReorderArgs(args))
	o.Finalize()

	rt, err := NewRuntime(o)
	if err != nil {
		color.Red("❌ %v\n", err)
		return 1
	}
	goal, err := CollectInput(fs.Args(), file, 256*1024)
	if err != nil {
		return rt.fatalErr(err)
	}
	if !o.Quiet {
		color.Cyan("\n🤖 ━━ Autonomous Agent Initiated ━━ (provider: %s, model: %s)\n\n", rt.ProvID, rt.Model)
	}
	code, err := rt.RunAgent(context.Background(), goal)
	if err != nil {
		return rt.fatalErr(err)
	}
	return code
}

func runGenCmd(args []string, mode string) int {
	var o Options
	fs := newFlagSet(mode, &o)
	file, _ := parseInputFlags(fs)
	_ = fs.Parse(ReorderArgs(args))
	o.Finalize()

	rt, err := NewRuntime(o)
	if err != nil {
		color.Red("❌ %v\n", err)
		return 1
	}
	input, err := CollectInput(fs.Args(), file, 512*1024)
	if err != nil {
		return rt.fatalErr(err)
	}

	ctx := context.Background()
	var code int
	var rerr error
	switch mode {
	case "plan":
		code, rerr = rt.RunPlan(ctx, input)
	case "debug":
		code, rerr = rt.RunDebug(ctx, input)
	case "doc":
		code, rerr = rt.RunDoc(ctx, input)
	}
	if rerr != nil {
		return rt.fatalErr(rerr)
	}
	return code
}

func runAudit(args []string) int {
	var o Options
	fs := newFlagSet("audit", &o)
	_ = fs.Parse(args)
	o.Finalize()
	target := "."
	if rest := fs.Args(); len(rest) > 0 {
		target = strings.Join(rest, " ")
	}
	fmt.Print(RunFullAuditReport(context.Background(), target))
	return 0
}

func runSecCmd(args []string) int {
	var o Options
	fs := newFlagSet("sec", &o)
	_ = fs.Parse(args)
	o.Finalize()
	return SecReport(context.Background(), fs.Args())
}

// ── config / provider / models / sessions ────────────────────────────────

func runConfigCmd(args []string) int {
	if len(args) == 0 {
		args = []string{"wizard"}
	}
	sub := strings.ToLower(args[0])
	rest := args[1:]

	switch sub {
	case "wizard":
		cfg, err := config.LoadConfig()
		if err != nil {
			color.Yellow("⚠️ %v", err)
		}
		if err := tui.RunConfigWizard(cfg); err != nil {
			return 1
		}
		return 0
	case "path":
		cfg, _ := config.LoadConfig()
		fmt.Println(cfg.ConfigPath)
		return 0
	case "show":
		cfg, err := config.LoadConfig()
		if err != nil {
			color.Red("%v", err)
			return 1
		}
		printConfigSummary(cfg)
		return 0
	case "set":
		cfg, err := config.LoadConfig()
		if err != nil {
			color.Red("%v", err)
			return 1
		}
		// Accept both `k=v` and `k v` forms.
		var pairs []string
		for i := 0; i < len(rest); i++ {
			arg := rest[i]
			if strings.Contains(arg, "=") {
				pairs = append(pairs, arg)
			} else if i+1 < len(rest) {
				i++
				pairs = append(pairs, arg+"="+rest[i])
			} else {
				color.Red("❌ expected key=value or key value, got %q", arg)
				return 2
			}
		}
		for _, kv := range pairs {
			if err := applyConfigSet(cfg, kv); err != nil {
				color.Red("❌ %v", err)
				return 2
			}
			fmt.Printf("✅ set %s\n", kv)
		}
		if err := cfg.Save(); err != nil {
			color.Red("❌ %v", err)
			return 1
		}
		fmt.Printf("Saved to %s (0600)\n", cfg.ConfigPath)
		return 0
	}
	color.Red("Unknown config subcommand %q (want: wizard | show | path | set k=v ...)\n", sub)
	return 2
}

// applyConfigSet accepts "<provider>.<field>=<value>" and global keys
// (provider, temp, stream, autosave, approve_mode, max_context_tokens,
// agent_max_steps, request_timeout_sec, max_retries).
func applyConfigSet(cfg *config.Config, kv string) error {
	eq := strings.Index(kv, "=")
	if eq < 0 {
		return fmt.Errorf("expected key=value, got %q", kv)
	}
	key, val := strings.TrimSpace(kv[:eq]), strings.TrimSpace(kv[eq+1:])

	if dot := strings.Index(key, "."); dot > 0 {
		provPart, field := key[:dot], key[dot+1:]
		id, ok := provider.NormalizeID(provPart)
		if !ok {
			return fmt.Errorf("unknown provider %q", provPart)
		}
		p := cfg.Providers[config.ProviderType(id)]
		switch field {
		case "api_key", "key":
			p.APIKey = val
		case "model":
			p.Model = val
		case "base_url", "url":
			p.BaseURL = strings.TrimSuffix(val, "/")
		default:
			return fmt.Errorf("unknown provider field %q (want api_key|model|base_url)", field)
		}
		if cfg.Providers == nil {
			cfg.Providers = map[config.ProviderType]config.ProviderConfig{}
		}
		cfg.Providers[config.ProviderType(id)] = p
		return nil
	}

	switch key {
	case "provider":
		id, ok := provider.NormalizeID(val)
		if !ok {
			return fmt.Errorf("unknown provider %q", val)
		}
		cfg.ActiveProvider = config.ProviderType(id)
	case "temp":
		var v float64
		if _, err := fmt.Sscanf(val, "%f", &v); err != nil || v < 0 || v > 2 {
			return fmt.Errorf("temp must be 0..2")
		}
		cfg.DefaultTemp = v
	case "stream":
		b, err := parseBool(val)
		if err != nil {
			return err
		}
		cfg.StreamResponses = b
	case "autosave":
		b, err := parseBool(val)
		if err != nil {
			return err
		}
		cfg.AutoSaveHistory = b
	case "context_budget":
		// alias
		var v int
		if _, err := fmt.Sscanf(val, "%d", &v); err != nil || v < 0 {
			return fmt.Errorf("context_budget must be a non-negative integer")
		}
		cfg.MaxContextTokens = v
	case "approve_mode":
		switch val {
		case "ask", "yolo", "read-only":
			cfg.ApproveMode = val
		default:
			return fmt.Errorf("approve_mode must be ask|yolo|read-only")
		}
	case "max_context_tokens":
		var v int
		if _, err := fmt.Sscanf(val, "%d", &v); err != nil || v < 0 {
			return fmt.Errorf("max_context_tokens must be a non-negative integer")
		}
		cfg.MaxContextTokens = v
	case "agent_max_steps":
		var v int
		if _, err := fmt.Sscanf(val, "%d", &v); err != nil || v <= 0 || v > 100 {
			return fmt.Errorf("agent_max_steps must be 1..100")
		}
		cfg.AgentMaxSteps = v
	case "request_timeout_sec":
		var v int
		if _, err := fmt.Sscanf(val, "%d", &v); err != nil || v < 1 {
			return fmt.Errorf("request_timeout_sec must be >= 1")
		}
		cfg.RequestTimeoutSec = v
	case "max_retries":
		var v int
		if _, err := fmt.Sscanf(val, "%d", &v); err != nil || v < 0 || v > 10 {
			return fmt.Errorf("max_retries must be 0..10")
		}
		cfg.MaxRetries = &v
	default:
		return fmt.Errorf("unknown key %q (try <provider>.<field> or provider/temp/stream/approve_mode/...)", key)
	}
	return nil
}

func parseBool(v string) (bool, error) {
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	}
	return false, fmt.Errorf("expected boolean value, got %q", v)
}

func printConfigSummary(cfg *config.Config) {
	fmt.Printf("Config file : %s\n", cfg.ConfigPath)
	fmt.Printf("Provider    : %s\n", cfg.ActiveProvider)
	fmt.Printf("Temperature : %.2f\n", cfg.DefaultTemp)
	fmt.Printf("Streaming   : %v\n", cfg.StreamResponses)
	fmt.Printf("Autosave    : %v\n", cfg.AutoSaveHistory)
	fmt.Printf("Approve     : %s\n", orDefault(cfg.ApproveMode, "ask"))
	fmt.Printf("Ctx budget  : %d tokens (0 = unlimited)\n", cfg.MaxContextTokens)
	fmt.Println()
	fmt.Println("Providers with stored credentials (keys are masked):")
	for _, d := range provider.All() {
		p := cfg.Providers[config.ProviderType(d.ID)]
		if p.APIKey == "" && p.Model == d.DefaultModel && p.BaseURL == d.DefaultBaseURL {
			continue // untouched default entry
		}
		active := ""
		if string(cfg.ActiveProvider) == d.ID {
			active = " (ACTIVE)"
		}
		fmt.Printf("  %-12s key=%-18s model=%-40s%s\n", d.ID, config.MaskKey(p.APIKey), orDefault(p.Model, "-"), active)
	}
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func runProviderCmd(args []string) int {
	sub := "list"
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}
	rest := args[1:]

	switch sub {
	case "list":
		cfg, _ := config.LoadConfig()
		fmt.Printf("%-13s %-24s %-9s %-6s %s\n", "ID", "NAME", "KEY ENV", "ACTIVE", "DEFAULT MODEL")
		fmt.Println(strings.Repeat("─", 100))
		for _, d := range provider.All() {
			active := ""
			if string(cfg.ActiveProvider) == d.ID {
				active = "  ✓"
			}
			keyEnv := d.EnvPrefix + "_API_KEY"
			if len(d.KeyEnvs) > 0 {
				keyEnv = keyEnv + " | " + strings.Join(d.KeyEnvs, " | ")
			}
			if !d.RequiresKey {
				keyEnv = "(optional)"
			}
			model := d.DefaultModel
			if p, ok := cfg.Providers[config.ProviderType(d.ID)]; ok && p.Model != "" {
				model = p.Model
			}
			fmt.Printf("%-13s %-24s %-9s %-6s %s\n", d.ID, d.DisplayName, keyEnv, active, model)
		}
		fmt.Println("\nConfigure a key:  shellsage config set <id>.api_key <KEY>")
		return 0

	case "show":
		cfg, _ := config.LoadConfig()
		id := string(cfg.ActiveProvider)
		if len(rest) > 0 {
			id = rest[0]
		}
		d, ok := provider.ByID(id)
		if !ok {
			color.Red("Unknown provider %q", id)
			return 2
		}
		p := cfg.Providers[config.ProviderType(d.ID)]
		fmt.Printf("Provider   : %s (%s)\n", d.ID, d.DisplayName)
		fmt.Printf("Protocol   : %s\n", d.Format)
		fmt.Printf("Base URL   : %s\n", orDefault(p.BaseURL, d.DefaultBaseURL))
		fmt.Printf("Model      : %s\n", orDefault(p.Model, d.DefaultModel))
		fmt.Printf("API key    : %s\n", config.MaskKey(p.APIKey))
		fmt.Printf("Key env    : %s_API_KEY", d.EnvPrefix)
		if len(d.KeyEnvs) > 0 {
			fmt.Printf("  (also: %s)", strings.Join(d.KeyEnvs, ", "))
		}
		fmt.Println()
		if d.DocsURL != "" {
			fmt.Printf("Keys/docs    : %s\n", d.DocsURL)
		}
		fmt.Printf("Quick models : %s\n", strings.Join(d.Models, ", "))
		return 0

	case "use":
		if len(rest) == 0 {
			color.Red("Usage: shellsage provider use <id>")
			return 2
		}
		cfg, err := config.LoadConfig()
		if err != nil {
			color.Red("%v", err)
			return 1
		}
		id, ok := provider.NormalizeID(rest[0])
		if !ok {
			color.Red("Unknown provider %q — run `shellsage provider list`", rest[0])
			return 2
		}
		cfg.ActiveProvider = config.ProviderType(id)
		if err := cfg.Save(); err != nil {
			color.Red("%v", err)
			return 1
		}
		p, err := cfg.NewActiveProvider()
		if err == nil {
			fmt.Printf("✅ Active provider: %s (%s)\n", id, p.Name())
		} else {
			fmt.Printf("✅ Active provider: %s\n", id)
		}
		return 0
	}

	color.Red("Unknown provider subcommand %q (list|show|use)", sub)
	return 2
}

func runModelsCmd(args []string) int {
	sub := "list"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = strings.ToLower(args[0])
		args = args[1:]
	}

	var o Options
	fs := newFlagSet("models", &o)
	_ = fs.Parse(args)
	o.Finalize()

	cfg, err := config.LoadConfig()
	if err != nil {
		color.Red("%v", err)
		return 1
	}
	if o.Provider != "" {
		id, _ := provider.NormalizeID(o.Provider)
		cfg.ActiveProvider = config.ProviderType(id)
	}

	switch sub {
	case "list":
		if o.Remote || len(fs.Args()) > 0 && strings.EqualFold(fs.Args()[0], "--remote") {
			rt, err := NewRuntime(o)
			if err != nil {
				color.Red("%v", err)
				return 1
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			models, err := listRemote(ctx, rt.Prov)
			if err != nil {
				color.Red("❌ remote model list failed: %v", err)
				return 1
			}
			fmt.Printf("Live models for %s (%d):\n", cfg.ActiveProvider, len(models))
			for i, m := range models {
				fmt.Printf("  [%3d] %s\n", i+1, m)
			}
			return 0
		}
		rt, err := NewRuntime(o)
		if err != nil {
			color.Red("%v", err)
			return 1
		}
		models := rt.Prov.ListAvailableModels()
		fmt.Printf("Quick-pick models for %s (default: %s):\n", cfg.ActiveProvider, rt.Model)
		for i, m := range models {
			marker := "  "
			if m == rt.Model {
				marker = "→ "
			}
			fmt.Printf("  %s[%2d] %s\n", marker, i+1, m)
		}
		fmt.Println("\nTip: `models list --remote` asks the provider API for the live catalog;")
		fmt.Println("     `models use <id>` sets the default. Any model id works even if not listed.")
		return 0

	case "use":
		rest := fs.Args()
		if len(rest) == 0 {
			color.Red("Usage: shellsage models use <model-id>")
			return 2
		}
		if err := cfg.SetProviderModel(string(cfg.ActiveProvider), rest[0]); err != nil {
			color.Red("%v", err)
			return 2
		}
		if err := cfg.Save(); err != nil {
			color.Red("%v", err)
			return 1
		}
		fmt.Printf("✅ Default model for %s → %s\n", cfg.ActiveProvider, rest[0])
		return 0
	}
	color.Red("Unknown models subcommand %q (list|use)", sub)
	return 2
}

func listRemote(ctx context.Context, p provider.Provider) ([]string, error) {
	if l, ok := p.(provider.RemoteModelLister); ok {
		return l.ListRemoteModels(ctx)
	}
	if f, ok := p.(provider.ModelFetcher); ok {
		return f.FetchInstalledModels(), nil
	}
	return nil, fmt.Errorf("provider %s does not support live model listing", p.Name())
}

func runSessionsCmd(args []string) int {
	sub := "list"
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}
	rest := args[1:]

	switch sub {
	case "list":
		list := history.ListSavedConversations()
		if len(list) == 0 {
			color.Yellow("No saved conversations (save one from the REPL with /save, or sessions live in %s).\n", history.GetConversationsDir())
			return 0
		}
		fmt.Printf("%-34s %-40s %-6s %-8s %s\n", "FILE", "TITLE", "TURNS", "TOKENS", "UPDATED")
		for _, c := range list {
			title := c.Title
			if len(title) > 38 {
				title = title[:38] + "…"
			}
			fmt.Printf("%-34s %-40s %-6d %-8d %s\n", c.Filename, title, c.MessageCount, c.TotalTokens, c.UpdatedAt.Format("2006-01-02 15:04"))
		}
		return 0

	case "delete":
		if len(rest) == 0 {
			color.Red("Usage: shellsage sessions delete <file>")
			return 2
		}
		path, err := history.DeleteTree(rest[0])
		if err != nil {
			color.Red("❌ %v", err)
			return 1
		}
		fmt.Printf("🗑  deleted %s\n", path)
		return 0

	case "rename":
		if len(rest) < 2 {
			color.Red("Usage: shellsage sessions rename <old> <new>")
			return 2
		}
		path, err := history.RenameTree(rest[0], rest[1])
		if err != nil {
			color.Red("❌ %v", err)
			return 1
		}
		fmt.Printf("✏️  renamed to %s\n", path)
		return 0

	case "load", "resume":
		if len(rest) == 0 {
			color.Red("Usage: shellsage sessions load <file>   (prints path; resume inside REPL with /load)")
			return 2
		}
		tree, err := history.LoadTree(rest[0])
		if err != nil {
			color.Red("❌ %v", err)
			return 1
		}
		fmt.Printf("✅ '%s' — %d nodes on active path. Resume interactively with: shellsage chat  then  /load %s\n", tree.Title, len(tree.GetActivePath()), rest[0])
		return 0
	}

	color.Red("Unknown sessions subcommand %q (list|delete|rename|load)", sub)
	return 2
}

// ── doctor ────────────────────────────────────────────────────────────────

type check struct {
	name   string
	ok     bool
	warn   bool
	detail string
}

func runDoctor(args []string) int {
	var o Options
	fs := newFlagSet("doctor", &o)
	_ = fs.Parse(args)
	o.Finalize()

	var checks []check
	failures := 0

	add := func(name string, ok bool, detail string) {
		checks = append(checks, check{name: name, ok: ok, detail: detail})
		if !ok {
			failures++
		}
	}
	addWarn := func(name, detail string) {
		checks = append(checks, check{name: name, ok: true, warn: true, detail: detail})
	}

	// 1. Config file
	cfg, err := config.LoadConfig()
	if o.ConfigPath != "" {
		cfg, err = config.LoadConfigAt(o.ConfigPath)
	}
	if err != nil {
		add("config load", false, err.Error())
	} else {
		if fi, statErr := os.Stat(cfg.ConfigPath); statErr != nil {
			addWarn("config file", cfg.ConfigPath+" — not saved yet (using defaults; first `config set`/wizard will create it with 0600)")
		} else {
			mode := fmt.Sprintf("%04o", fi.Mode().Perm())
			detail := fmt.Sprintf("%s (mode %s)", cfg.ConfigPath, mode)
			if mode == "0644" || mode == "0664" || mode == "0666" {
				addWarn("config perms", detail+" — world-readable config holding API keys! `chmod 600` it")
			} else {
				add("config file", true, detail)
			}
		}
	}

	// 2. Provider selection & key (honor --provider override like every other command)
	activeID := string(cfg.ActiveProvider)
	if o.Provider != "" {
		if pid, ok := provider.NormalizeID(o.Provider); ok {
			activeID = pid
		}
	}
	d, dOK := provider.ByID(activeID)
	if !dOK {
		add("active provider", false, fmt.Sprintf("%q not in registry", activeID))
	} else {
		p := cfg.Providers[config.ProviderType(activeID)]
		if d.RequiresKey && p.APIKey == "" {
			envHint := d.EnvPrefix + "_API_KEY"
			if len(d.KeyEnvs) > 0 {
				envHint = fmt.Sprintf("%s (or %s)", envHint, strings.Join(d.KeyEnvs, ", "))
			}
			add("provider api key", false, fmt.Sprintf("%s has no key — set it (config set %s.api_key … / %s)", d.DisplayName, d.ID, envHint))
		} else if d.RequiresKey {
			add("provider api key", true, fmt.Sprintf("%s: %s", d.DisplayName, config.MaskKey(p.APIKey)))
		} else {
			add("provider api key", true, d.DisplayName+" — local provider, key optional")
		}

		// 3. Endpoint reachability + model check
		rt, rerr := NewRuntime(o)
		if rerr != nil {
			add("provider init", false, rerr.Error())
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			start := time.Now()
			models, lerr := listRemote(ctx, rt.Prov)
			if lerr != nil {
				add("endpoint reachable", false, lerr.Error())
			} else {
				add("endpoint reachable", true, fmt.Sprintf("%d models listed in %v", len(models), time.Since(start).Round(time.Millisecond)))
				if rt.Model != "" && len(models) > 0 {
					found := false
					for _, m := range models {
						if m == rt.Model {
							found = true
							break
						}
					}
					if !found {
						addWarn("model available", fmt.Sprintf("active model %q not in provider list — it may still work (some APIs don't enumerate deployments)", rt.Model))
					} else {
						add("model available", true, rt.Model)
					}
				}
			}
		}
	}

	// 4. Clipboard
	if _, cerr := clipboard.ReadAll(); cerr != nil {
		addWarn("clipboard", "unavailable (headless session?) — /copy will fail: "+cerr.Error())
	} else {
		add("clipboard", true, "OS clipboard reachable")
	}

	// 5. Configured providers summary
	configured := 0
	for _, dd := range provider.All() {
		if p := cfg.Providers[config.ProviderType(dd.ID)]; p.APIKey != "" {
			configured++
		}
	}
	add("providers with keys", configured > 0, fmt.Sprintf("%d configured", configured))

	// Print report
	fmt.Println()
	color.New(color.FgCyan, color.Bold).Printf("🩺 ShellSage Doctor — environment health\n")
	color.HiBlack(strings.Repeat("─", 64) + "\n")
	for _, c := range checks {
		icon := color.GreenString("✓")
		if !c.ok {
			icon = color.RedString("✗")
		} else if c.warn {
			icon = color.YellowString("!")
		}
		fmt.Printf(" %s %-22s %s\n", icon, c.name, c.detail)
	}
	fmt.Println()
	if failures > 0 {
		color.Red("❗ %d problem(s) detected.\n", failures)
		return 1
	}
	color.Green("✅ All checks passed.\n")
	return 0
}

// ── completion ────────────────────────────────────────────────────────────

func runCompletion(args []string) int {
	shell := "bash"
	if len(args) > 0 {
		shell = strings.ToLower(args[0])
	}
	cmds := strings.Join(Commands, " ")
	switch shell {
	case "bash":
		fmt.Printf(`_shellsage() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  local cmds="%s"
  if [ "$COMP_CWORD" -eq 1 ]; then
    COMPREPLY=( $(compgen -W "${cmds} --help --version" -- "${cur}") )
  else
    case "${COMP_WORDS[1]}" in
      provider) COMPREPLY=( $(compgen -W "list show use" -- "$cur") ) ;;
      config)   COMPREPLY=( $(compgen -W "wizard show path set" -- "$cur") ) ;;
      models)   COMPREPLY=( $(compgen -W "list use" -- "$cur") ) ;;
      sessions) COMPREPLY=( $(compgen -W "list delete rename load" -- "$cur") ) ;;
      sec)      COMPREPLY=( $(compgen -W "headers ssl ports sast audit" -- "$cur") ) ;;
      completion) COMPREPLY=( $(compgen -W "bash zsh powershell pwsh fish" -- "$cur") ) ;;
      *)        COMPREPLY=( $(compgen -W "--json --quiet --yes --provider --model --stream --temp --steps --timeout --image --file" -- "$cur") ) ;;
    esac
  fi
}
complete -F _shellsage shellsage
`, cmds)
	case "zsh":
		var pairs []string
		descs := map[string]string{
			"chat": "interactive REPL", "ask": "one-shot chat", "agent": "autonomous agent run",
			"plan": "implementation plan", "debug": "root-cause diagnosis", "doc": "documentation generator",
			"audit": "full security audit", "sec": "security checks (headers|ssl|ports|sast|audit)",
			"config": "config wizard/show/path/set", "provider": "provider list/show/use",
			"models": "models list/use", "sessions": "saved sessions list/delete/rename",
			"doctor": "environment diagnostics", "completion": "shell completion script",
			"version": "print version", "help": "show usage",
		}
		for _, c := range Commands {
			pairs = append(pairs, fmt.Sprintf("'%s:%s'", c, descs[c]))
		}
		fmt.Printf(`#compdef shellsage
_shellsage() {
  local -a commands
  commands=(%s)
  _describe 'command' commands
}
compdef _shellsage shellsage
`, strings.Join(pairs, " "))
	case "fish":
		for _, c := range Commands {
			fmt.Printf("complete -c shellsage -n __fish_use_subcommand -a %s\n", c)
		}
	case "powershell", "pwsh", "ps1":
		fmt.Printf(`Register-ArgumentCompleter -Native -CommandName shellsage -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)
    $subcommands = @("%s")
    $subcommands | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
    }
}
`, strings.Join(Commands, `", "`))
	default:
		color.Red("Unknown shell %q (bash|zsh|fish|powershell)", shell)
		return 2
	}
	return 0
}

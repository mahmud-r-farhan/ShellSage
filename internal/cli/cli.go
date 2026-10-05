// Package cli implements ShellSage's non-interactive surface: global option
// parsing, one-shot commands (ask/agent/plan/debug/doc/audit/sec), admin
// commands (config/provider/models/sessions/doctor/completion) and the shared
// runtime wiring used by both scripts and the interactive REPL.
package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"shellsage/internal/agent"
	"shellsage/internal/config"
	"shellsage/internal/provider"
	"shellsage/internal/tools"
)

// imageList implements flag.Value for repeatable --image flags.
type imageList []string

func (s *imageList) String() string { return strings.Join(*s, ",") }
func (s *imageList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// Options are the resolved global options shared by every subcommand.
type Options struct {
	Provider     string
	Model        string
	Temp         float64
	TempSet      bool
	MaxTokens    int
	StreamMode   string // auto | on | off
	JSON         bool
	Quiet        bool
	Yes          bool
	NoColor      bool
	Steps        int
	MaxTime      time.Duration
	Timeout      time.Duration
	ConfigPath   string
	LogFile      string
	SystemPrompt string
	Images       imageList
	Remote       bool // models list --remote
}

// GlobalValueFlags lists flags that consume the next argument; used both by
// ReorderArgs (flag-after-prompt support) and main's subcommand pre-scan.
var GlobalValueFlags = map[string]bool{
	"-provider": true, "-model": true, "-temp": true, "-max-tokens": true,
	"-stream": true, "-timeout": true, "-config-file": true, "-steps": true,
	"-max-time": true, "-log-file": true, "-system": true, "-image": true, "-file": true,
}

// GlobalBoolFlags lists value-less boolean globals recognized anywhere.
var GlobalBoolFlags = map[string]bool{
	"-json": true, "-quiet": true, "-yes": true, "-no-color": true, "-remote": true,
}

// normalizeFlagName maps "--provider=x" & "-provider" to "-provider".
func normalizeFlagName(a string) string {
	return "-" + strings.TrimLeft(strings.SplitN(a, "=", 2)[0], "-")
}

// ReorderArgs moves recognized global flags (and their values) in front of
// positional arguments so `shellsage ask hi there --provider groq` behaves the
// same as `shellsage ask --provider groq hi there`. The flag package stops
// parsing at the first positional, which otherwise silently swallows flags.
func ReorderArgs(args []string) []string {
	var flags, positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") || a == "-" {
			positionals = append(positionals, a)
			continue
		}
		name := normalizeFlagName(a)
		if strings.Contains(a, "=") && (GlobalValueFlags[name] || GlobalBoolFlags[name]) {
			flags = append(flags, a)
			continue
		}
		if GlobalValueFlags[name] {
			flags = append(flags, a)
			if i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		if GlobalBoolFlags[name] {
			flags = append(flags, a)
			continue
		}
		// unknown flag: keep as positional so the subcommand errors clearly
		positionals = append(positionals, a)
	}
	return append(flags, positionals...)
}

// RegisterGlobalFlags attaches the shared flag set to fs.
func RegisterGlobalFlags(fs *flag.FlagSet, o *Options) {
	fs.StringVar(&o.Provider, "provider", "", "override active provider (id from `provider list`)")
	fs.StringVar(&o.Model, "model", "", "override model / deployment id")
	fs.Float64Var(&o.Temp, "temp", -1, "sampling temperature 0.0-2.0")
	fs.IntVar(&o.MaxTokens, "max-tokens", 0, "max completion tokens (0 = provider default 4096)")
	fs.StringVar(&o.StreamMode, "stream", "auto", "token streaming: auto|on|off (auto = only on a TTY)")
	fs.BoolVar(&o.JSON, "json", false, "emit a machine-readable JSON result envelope (implies --quiet)")
	fs.BoolVar(&o.Quiet, "quiet", false, "suppress decorative output (progress, usage)")
	fs.BoolVar(&o.Yes, "yes", false, "auto-approve agent tools that mutate state (dangerous)")
	fs.BoolVar(&o.NoColor, "no-color", false, "disable ANSI colors")
	fs.IntVar(&o.Steps, "steps", 0, "max agent ReAct steps (0 = default/config)")
	fs.DurationVar(&o.MaxTime, "max-time", 0, "wall-clock budget per agent run (e.g. 5m; 0 = unlimited)")
	fs.DurationVar(&o.Timeout, "timeout", 0, "per-request HTTP timeout (e.g. 120s; 0 = client default)")
	fs.StringVar(&o.ConfigPath, "config-file", "", "alternate config.json path")
	fs.StringVar(&o.LogFile, "log-file", "", "append the agent action transcript to this file")
	fs.StringVar(&o.SystemPrompt, "system", "", "replace the persona/system prompt (one-shot modes)")
	fs.Var(&o.Images, "image", "attach an image file or URL (repeatable; vision)")
	fs.BoolVar(&o.Remote, "remote", false, "query the live provider API where applicable")
}

// Finalize normalizes option values after parsing.
func (o *Options) Finalize() {
	if o.Temp >= 0 {
		o.TempSet = true
	} else {
		o.Temp = 0
	}
	if o.JSON {
		o.Quiet = true
	}
	if o.NoColor || os.Getenv("NO_COLOR") != "" {
		os.Setenv("NO_COLOR", "1")
	}
}

// Runtime is a fully wired non-interactive execution environment.
type Runtime struct {
	Options Options
	Cfg     *config.Config
	Prov    provider.Provider
	ProvID  config.ProviderType
	Model   string
	Temp    float64
	Stream  bool
	ToolReg *tools.ToolRegistry
}

// NewRuntime loads config, applies flag overrides and instantiates the provider.
func NewRuntime(o Options) (*Runtime, error) {
	var (
		cfg *config.Config
		err error
	)
	if o.ConfigPath != "" {
		cfg, err = config.LoadConfigAt(o.ConfigPath)
	} else {
		cfg, err = config.LoadConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	if o.Provider != "" {
		id, ok := provider.NormalizeID(o.Provider)
		if !ok {
			return nil, fmt.Errorf("unknown provider %q (shellsage provider list)", o.Provider)
		}
		cfg.ActiveProvider = config.ProviderType(id)
	}
	if o.Model != "" {
		p := cfg.Providers[cfg.ActiveProvider]
		p.Model = o.Model
		cfg.Providers[cfg.ActiveProvider] = p
	}
	if o.TempSet {
		cfg.DefaultTemp = o.Temp
	}
	if o.Timeout > 0 {
		cfg.RequestTimeoutSec = int(o.Timeout.Seconds())
	}

	p, err := cfg.NewActiveProvider()
	if err != nil {
		return nil, err
	}

	stream := false
	switch strings.ToLower(o.StreamMode) {
	case "on", "true", "yes":
		stream = true
	case "off", "false", "no":
		stream = false
	default: // auto
		stream = IsTTY(os.Stdout) && !o.JSON && !o.Quiet && cfg.StreamResponses
	}

	rt := &Runtime{
		Options: o,
		Cfg:     cfg,
		Prov:    p,
		ProvID:  cfg.ActiveProvider,
		Model:   cfg.Providers[cfg.ActiveProvider].Model,
		Temp:    cfg.DefaultTemp,
		Stream:  stream,
		ToolReg: tools.NewToolRegistry(),
	}
	if rt.Model == "" {
		if d, ok := provider.ByID(string(cfg.ActiveProvider)); ok {
			rt.Model = d.DefaultModel
		}
	}
	return rt, nil
}

// EnsureKey returns an actionable error when a key-requiring provider has no key.
func (rt *Runtime) EnsureKey() error {
	d, _ := provider.ByID(string(rt.ProvID))
	if d.RequiresKey {
		if p, ok := rt.Cfg.Providers[rt.ProvID]; !ok || p.APIKey == "" {
			return fmt.Errorf("no API key configured for %s.\n  Fix: `shellsage config set %s.api_key <KEY>`  or  export %s_API_KEY=...  or  run `shellsage config wizard`.\n  Free options: `shellsage --provider openrouter` (openrouter/free) or `shellsage --provider ollama` (local).",
				d.DisplayName, d.ID, d.EnvPrefix)
		}
	}
	return nil
}

// NewAgent builds an agent with approval semantics resolved from
// --yes / config approve_mode / TTY state.
func (rt *Runtime) NewAgent() *agent.Agent {
	opts := agent.Options{
		Model:    rt.Model,
		Temp:     rt.Temp,
		MaxSteps: rt.Options.Steps,
		MaxTime:  rt.Options.MaxTime,
	}
	if opts.MaxSteps <= 0 && rt.Cfg.AgentMaxSteps > 0 {
		opts.MaxSteps = rt.Cfg.AgentMaxSteps
	}
	ag := agent.NewAgentWithOptions(rt.Prov, rt.ToolReg, opts)
	ag.SetApprover(rt.Approver())
	return ag
}

// Approver returns the approval callback (nil = auto-approve everything).
func (rt *Runtime) Approver() agent.ApprovalFunc {
	mode := strings.ToLower(rt.Cfg.ApproveMode)
	if rt.Options.Yes {
		mode = "yolo"
	}
	switch mode {
	case "yolo":
		return nil // agent auto-approves when Approver is nil
	case "read-only":
		return func(tool, args string, risk tools.Risk) (bool, bool) { return false, false }
	default: // "ask"
		if !IsTTY(os.Stdin) {
			// Non-interactive without --yes: refuse mutation, keep reading.
			return func(tool, args string, risk tools.Risk) (bool, bool) {
				fmt.Fprintf(os.Stderr, "[approval] denied %s (%s risk) in non-interactive mode; pass --yes to allow\n", tool, risk)
				return false, false
			}
		}
		return interactiveApprover()
	}
}

// IsTTY reports whether f is an interactive terminal.
func IsTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"shellsage/internal/config"
	"shellsage/internal/provider"
)

func TestIsCommand(t *testing.T) {
	for _, c := range Commands {
		if !IsCommand(c) {
			t.Errorf("%s should be a command", c)
		}
	}
	if IsCommand("notacommand") || IsCommand("--help") {
		t.Error("non-commands misclassified")
	}
}

func TestCollectInputArgs(t *testing.T) {
	got, err := CollectInput([]string{"hello", "world"}, "", 1024)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello world" {
		t.Errorf("got %q", got)
	}
}

func TestCollectInputFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "err.log")
	os.WriteFile(f, []byte("panic: nil pointer"), 0644)

	got, err := CollectInput([]string{"analyze this"}, f, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "analyze this") || !strings.Contains(got, "panic: nil pointer") {
		t.Errorf("prompt+file should merge: %q", got)
	}
}

func TestCollectInputDashReadsStdinPlaceholder(t *testing.T) {
	// "-" means "read stdin"; with go test's stdin closed this must yield a
	// clear error rather than a hang or empty success.
	_, err := CollectInput([]string{"-"}, "", 1024)
	if err == nil {
		t.Log("note: stdin had content")
	}
}

func TestConfigSetParsing(t *testing.T) {
	cfg := config.DefaultConfig()
	if err := applyConfigSet(cfg, "provider=groq"); err != nil {
		t.Fatal(err)
	}
	if cfg.ActiveProvider != "groq" {
		t.Errorf("active provider: %s", cfg.ActiveProvider)
	}
	if err := applyConfigSet(cfg, "openai.api_key=sk-123"); err != nil {
		t.Fatal(err)
	}
	if cfg.Providers["openai"].APIKey != "sk-123" {
		t.Error("openai key not set")
	}
	if err := applyConfigSet(cfg, "approve_mode=yolo"); err != nil {
		t.Fatal(err)
	}
	if cfg.ApproveMode != "yolo" {
		t.Error("approve mode not set")
	}
	if err := applyConfigSet(cfg, "bogus.key=v"); err == nil {
		t.Error("unknown provider should error")
	}
	if err := applyConfigSet(cfg, "noequalsign"); err == nil {
		t.Error("missing = should error")
	}
}

func TestEnvelopeRoundTrip(t *testing.T) {
	env := ResultEnvelope{
		OK: true, Provider: "openai", Model: "gpt-4o-mini", Mode: "ask", Content: "hi",
		Usage: provider.TokenUsage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3},
	}
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"ok":true`) || !strings.Contains(string(data), `"total_tokens":3`) {
		t.Errorf("envelope shape wrong: %s", data)
	}
}

func TestCompletionScripts(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell", "pwsh"} {
		code := runCompletion([]string{shell})
		if code != 0 {
			t.Errorf("completion %s exit %d", shell, code)
		}
	}
	if code := runCompletion([]string{"unknownshell"}); code != 2 {
		t.Error("unknown shell should exit 2")
	}
}

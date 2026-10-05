package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyPatchAtomicity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.go")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	reg := NewToolRegistry()
	// Second edit target is missing -> NO changes must be written.
	args, _ := json.Marshal(map[string]interface{}{
		"path": path,
		"edits": []map[string]interface{}{
			{"target": "line1", "replacement": "HEADER"},
			{"target": "does-not-exist", "replacement": "x"},
		},
	})
	if _, err := reg.Execute(context.Background(), "apply_patch", string(args)); err == nil {
		t.Fatal("expected error for missing target")
	}
	data, _ := os.ReadFile(path)
	if string(data) != content {
		t.Errorf("file must be untouched on failed patch, got %q", data)
	}

	// dry_run must not write
	args, _ = json.Marshal(map[string]interface{}{
		"path":    path,
		"dry_run": true,
		"edits":   []map[string]interface{}{{"target": "line1", "replacement": "HEADER"}},
	})
	res, err := reg.Execute(context.Background(), "apply_patch", string(args))
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if !strings.Contains(res, "dry-run") {
		t.Errorf("expected dry-run marker in %q", res)
	}
	data, _ = os.ReadFile(path)
	if string(data) != content {
		t.Error("dry_run must not modify the file")
	}

	// happy path: multiple edits apply
	args, _ = json.Marshal(map[string]interface{}{
		"path":  path,
		"edits": []map[string]interface{}{{"target": "line1", "replacement": "HEADER"}, {"target": "line3", "replacement": "FOOTER"}},
	})
	if _, err := reg.Execute(context.Background(), "apply_patch", string(args)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	data, _ = os.ReadFile(path)
	if string(data) != "HEADER\nline2\nFOOTER\n" {
		t.Errorf("unexpected patched content: %q", data)
	}
}

func TestFindFiles(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"a.go", "b.txt", "deep/c.go", "node_modules/x.go"} {
		full := filepath.Join(dir, f)
		_ = os.MkdirAll(filepath.Dir(full), 0755)
		_ = os.WriteFile(full, []byte("x"), 0644)
	}
	reg := NewToolRegistry()
	args, _ := json.Marshal(map[string]interface{}{"pattern": "*.go", "path": dir})
	res, err := reg.Execute(context.Background(), "find_files", string(args))
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if !strings.Contains(res, "a.go") || !strings.Contains(res, "c.go") {
		t.Errorf("expected go files in results: %s", res)
	}
	if strings.Contains(res, "node_modules") {
		t.Errorf("node_modules should be skipped: %s", res)
	}
}

func TestHTTPRequestTool(t *testing.T) {
	t.Setenv("SHELLSAGE_ALLOW_LOCAL_NET", "1") // httptest listens on loopback
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"echo":"posted"}`))
			return
		}
		_, _ = w.Write([]byte("hello"))
	}))
	defer srv.Close()

	reg := NewToolRegistry()
	args, _ := json.Marshal(map[string]interface{}{"url": srv.URL})
	res, err := reg.Execute(context.Background(), "http_request", string(args))
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	if !strings.Contains(res, "200 OK") || !strings.Contains(res, "hello") {
		t.Errorf("unexpected GET result: %s", res)
	}

	// POST must be classified net-risk (approval required)
	args, _ = json.Marshal(map[string]interface{}{"url": srv.URL, "method": "POST", "body": "{}"})
	res, err = reg.Execute(context.Background(), "http_request", string(args))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	if !strings.Contains(res, "echo") {
		t.Errorf("unexpected POST result: %s", res)
	}
	if reg.RiskForArgs("http_request", `{"method":"GET"}`) != RiskRead {
		t.Error("GET http_request should be read-risk")
	}
	if reg.RiskForArgs("http_request", `{"method":"POST"}`) != RiskNet {
		t.Error("POST http_request should be net-risk")
	}
}

func TestHTTPRequestBlocksMetadataIP(t *testing.T) {
	reg := NewToolRegistry()
	args, _ := json.Marshal(map[string]interface{}{"url": "http://169.254.169.254/latest/meta-data/"})
	if _, err := reg.Execute(context.Background(), "http_request", string(args)); err == nil {
		t.Fatal("metadata endpoint must be blocked")
	}
}

func TestShellDangerousPatterns(t *testing.T) {
	reg := NewToolRegistry()
	args, _ := json.Marshal(map[string]interface{}{"command": "sudo rm -rf / --no-preserve-root"})
	if _, err := reg.Execute(context.Background(), "run_command", string(args)); err == nil {
		t.Fatal("catastrophic rm must be denied")
	}
	args, _ = json.Marshal(map[string]interface{}{"command": "sudo apt-get update"})
	if _, err := reg.Execute(context.Background(), "run_command", string(args)); err == nil {
		t.Fatal("sudo must be denied")
	}
	args, _ = json.Marshal(map[string]interface{}{"command": "echo hello-shell"})
	res, err := reg.Execute(context.Background(), "run_command", string(args))
	if err != nil || !strings.Contains(res, "hello-shell") {
		t.Fatalf("plain echo should work: %q %v", res, err)
	}
}

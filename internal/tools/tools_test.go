package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFSTools(t *testing.T) {
	tempDir := t.TempDir()
	reg := NewToolRegistry()

	filePath := filepath.Join(tempDir, "sample.txt")

	// Test write_file
	_, err := reg.Execute(context.Background(), "write_file", `{"path":"`+filepath.ToSlash(filePath)+`","content":"Hello ShellSage"}`)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Test read_file
	content, err := reg.Execute(context.Background(), "read_file", `{"path":"`+filepath.ToSlash(filePath)+`"}`)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if content != "Hello ShellSage" {
		t.Errorf("unexpected content: %s", content)
	}

	// Test edit_file
	_, err = reg.Execute(context.Background(), "edit_file", `{"path":"`+filepath.ToSlash(filePath)+`","target":"Hello","replacement":"Greetings"}`)
	if err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}

	edited, _ := os.ReadFile(filePath)
	if string(edited) != "Greetings ShellSage" {
		t.Errorf("unexpected edited content: %s", string(edited))
	}
}

func TestShellTool(t *testing.T) {
	reg := NewToolRegistry()
	res, err := reg.Execute(context.Background(), "run_command", `{"command":"echo test_ok"}`)
	if err != nil {
		t.Fatalf("unexpected error running echo: %v", err)
	}
	if !strings.Contains(res, "test_ok") {
		t.Errorf("output missing test_ok: %s", res)
	}
}

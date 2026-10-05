package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type patchEdit struct {
	Target      string `json:"target"`
	Replacement string `json:"replacement"`
	// All can replace every occurrence instead of just the first.
	All bool `json:"all,omitempty"`
}

type patchOp struct {
	Path   string      `json:"path"`
	Edits  []patchEdit `json:"edits"`
	DryRun bool        `json:"dry_run,omitempty"`
}

// registerPatchTools adds multi-edit atomic patching — the single highest-value
// upgrade for coding agents: one call can apply several hunks to one file with
// dry-run previews and all-or-nothing semantics.
func (r *ToolRegistry) registerPatchTools() {
	r.Register(ToolDef{
		Name:        "apply_patch",
		Description: "Atomically applies one or more find/replace edits to a single file. Supports dry_run to preview a diff first. Fails without touching the file if ANY target is missing.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":    map[string]interface{}{"type": "string", "description": "File to patch"},
				"dry_run": map[string]interface{}{"type": "boolean", "description": "When true, return the diff preview without writing"},
				"edits": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"target":      map[string]interface{}{"type": "string", "description": "Exact text to find"},
							"replacement": map[string]interface{}{"type": "string", "description": "Replacement text"},
							"all":         map[string]interface{}{"type": "boolean", "description": "Replace all occurrences"},
						},
						"required": []string{"target", "replacement"},
					},
				},
			},
			"required": []string{"path", "edits"},
		},
		Risk: RiskWrite,
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			raw, _ := json.Marshal(args)
			var op patchOp
			if err := json.Unmarshal(raw, &op); err != nil {
				return "", fmt.Errorf("invalid apply_patch payload: %w", err)
			}
			if op.Path == "" {
				op.Path, _ = args["path"].(string)
			}
			if op.Path == "" || len(op.Edits) == 0 {
				return "", fmt.Errorf("apply_patch requires 'path' and non-empty 'edits'")
			}

			data, err := os.ReadFile(op.Path)
			if err != nil {
				return "", fmt.Errorf("cannot read %s: %w", op.Path, err)
			}
			content := string(data)
			newContent := content
			var diffLines []string

			for i, e := range op.Edits {
				if e.Target == "" {
					return "", fmt.Errorf("edit %d: empty 'target'", i+1)
				}
				count := strings.Count(newContent, e.Target)
				if count == 0 {
					return "", fmt.Errorf("edit %d: target not found in %s — no changes applied (atomic)", i+1, op.Path)
				}
				if !e.All && count > 1 {
					// Ambiguous single-replace: accept when the first occurrence is unambiguous after
					// sequential application, but warn.
				}
				if e.All {
					newContent = strings.ReplaceAll(newContent, e.Target, e.Replacement)
					diffLines = append(diffLines, fmt.Sprintf("@@ edit %d: replacing %d occurrence(s)\n-%s\n+%s", i+1, count, firstLines(e.Target, 3), firstLines(e.Replacement, 3)))
				} else {
					newContent = strings.Replace(newContent, e.Target, e.Replacement, 1)
					diffLines = append(diffLines, fmt.Sprintf("@@ edit %d\n-%s\n+%s", i+1, firstLines(e.Target, 3), firstLines(e.Replacement, 3)))
				}
			}

			preview := fmt.Sprintf("apply_patch %s (%d edits):\n%s", op.Path, len(op.Edits), strings.Join(diffLines, "\n"))

			if op.DryRun {
				return preview + "\n\n[dry-run: file NOT modified]", nil
			}

			info, _ := os.Stat(op.Path)
			mode := os.FileMode(0644)
			if info != nil {
				mode = info.Mode().Perm()
			}
			dir := filepath.Dir(op.Path)
			if dir != "" && dir != "." {
				_ = os.MkdirAll(dir, 0755)
			}
			if err := os.WriteFile(op.Path, []byte(newContent), mode); err != nil {
				return "", fmt.Errorf("failed to write patched file: %w", err)
			}

			return fmt.Sprintf("%s\nSuccessfully patched %s (%d bytes -> %d bytes)", preview, op.Path, len(content), len(newContent)), nil
		},
	})
}

func firstLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = append(lines[:n], fmt.Sprintf("... (+%d more lines)", len(lines)-n))
	}
	for i, l := range lines {
		if len(l) > 120 {
			lines[i] = l[:120] + "..."
		}
	}
	return strings.Join(lines, "\n")
}

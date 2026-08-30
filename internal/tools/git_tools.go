package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func (r *ToolRegistry) registerGitTools() {
	r.Register(ToolDef{
		Name:        "git_status",
		Description: "Shows current git repository status, branch, and modified files",
		Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			cmd := exec.CommandContext(ctx, "git", "status", "--short")
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out

			if err := cmd.Run(); err != nil {
				return fmt.Sprintf("git status returned error: %s", out.String()), nil
			}

			res := strings.TrimSpace(out.String())
			if res == "" {
				return "Working tree clean (no modified files).", nil
			}
			return res, nil
		},
	})

	r.Register(ToolDef{
		Name:        "git_diff",
		Description: "Shows git diff changes in the local repository",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "Optional file path to limit diff"},
			},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			path, _ := args["path"].(string)
			cmdArgs := []string{"diff"}
			if path != "" {
				cmdArgs = append(cmdArgs, path)
			}

			cmd := exec.CommandContext(ctx, "git", cmdArgs...)
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out

			if err := cmd.Run(); err != nil {
				return fmt.Sprintf("git diff returned error: %s", out.String()), nil
			}

			res := strings.TrimSpace(out.String())
			if res == "" {
				return "No unstaged git diff changes.", nil
			}
			if len(res) > 20000 {
				res = res[:20000] + "\n\n... (diff truncated)"
			}
			return res, nil
		},
	})
}

func (r *ToolRegistry) registerDefaultTools() {
	r.registerFSTools()
	r.registerShellTools()
	r.registerWebSearchTools()
	r.registerWebScrapeTools()
	r.registerGitTools()
	r.registerSecurityTools()
}

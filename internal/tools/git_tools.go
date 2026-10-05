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
		Name:        "git_log",
		Description: "Shows recent git commit history (subject lines) for context on what changed lately",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"n":    map[string]interface{}{"type": "integer", "description": "Number of commits (default 15, max 100)"},
				"path": map[string]interface{}{"type": "string", "description": "Optional file/dir to limit history"},
			},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			n := 15
			if v, ok := args["n"].(float64); ok && v > 0 {
				n = int(v)
			}
			if n > 100 {
				n = 100
			}
			gitArgs := []string{"log", "--oneline", "-n", fmt.Sprint(n)}
			if path, _ := args["path"].(string); path != "" {
				gitArgs = append(gitArgs, "--", path)
			}
			cmd := exec.CommandContext(ctx, "git", gitArgs...)
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out
			if err := cmd.Run(); err != nil {
				return fmt.Sprintf("git log returned error: %s", strings.TrimSpace(out.String())), nil
			}
			res := strings.TrimSpace(out.String())
			if res == "" {
				return "No commits found (fresh repository?).", nil
			}
			return res, nil
		},
	})

	r.Register(ToolDef{
		Name:        "git_branch",
		Description: "Shows the current branch and all local branches with their last commit",
		Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			cmd := exec.CommandContext(ctx, "git", "branch", "-vv")
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out
			if err := cmd.Run(); err != nil {
				return fmt.Sprintf("git branch returned error: %s", strings.TrimSpace(out.String())), nil
			}
			res := strings.TrimSpace(out.String())
			if res == "" {
				return "No branches.", nil
			}
			return res, nil
		},
	})

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
	r.registerPatchTools()
	r.registerNetTools()
}

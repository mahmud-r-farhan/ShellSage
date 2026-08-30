package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func (r *ToolRegistry) registerShellTools() {
	r.Register(ToolDef{
		Name:        "run_command",
		Description: "Executes a shell command (e.g. go test, git, build, npm) and captures output",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{"type": "string", "description": "The command line string to execute"},
				"timeout": map[string]interface{}{"type": "integer", "description": "Timeout in seconds (default 30)"},
			},
			"required": []string{"command"},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			command, _ := args["command"].(string)
			if command == "" {
				command, _ = args["input"].(string)
			}
			if command == "" {
				return "", fmt.Errorf("missing 'command' argument")
			}

			// Block destructive system commands
			blocked := []string{"rm -rf /", "rmdir /s /q c:\\windows", "format c:"}
			for _, b := range blocked {
				if strings.Contains(strings.ToLower(command), b) {
					return "", fmt.Errorf("command blocked for safety: dangerous system command")
				}
			}

			timeoutSec := 30
			if t, ok := args["timeout"].(float64); ok && t > 0 {
				timeoutSec = int(t)
			}

			cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
			defer cancel()

			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				cmd = exec.CommandContext(cmdCtx, "cmd", "/C", command)
			} else {
				cmd = exec.CommandContext(cmdCtx, "sh", "-c", command)
			}

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			output := stdout.String()
			errOutput := stderr.String()

			var sb strings.Builder
			if output != "" {
				sb.WriteString("STDOUT:\n")
				sb.WriteString(output)
			}
			if errOutput != "" {
				if sb.Len() > 0 {
					sb.WriteString("\n")
				}
				sb.WriteString("STDERR:\n")
				sb.WriteString(errOutput)
			}

			res := sb.String()
			if len(res) > 30000 {
				res = res[:30000] + "\n... (output truncated)"
			}

			if err != nil {
				return fmt.Sprintf("Command failed with error (%v):\n%s", err, res), nil
			}

			if res == "" {
				return "(Command executed successfully with no output)", nil
			}

			return res, nil
		},
	})
}

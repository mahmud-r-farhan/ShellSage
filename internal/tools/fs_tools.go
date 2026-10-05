package tools

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func (r *ToolRegistry) registerFSTools() {
	// Read File
	r.Register(ToolDef{
		Name:        "read_file",
		Description: "Reads the content of a local file",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "Relative or absolute path to the file"},
			},
			"required": []string{"path"},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			path, _ := args["path"].(string)
			if path == "" {
				path, _ = args["input"].(string)
			}
			if path == "" {
				return "", fmt.Errorf("missing 'path' argument")
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("failed to read file '%s': %w", path, err)
			}

			// Truncate if excessively large
			if len(data) > 50000 {
				return string(data[:50000]) + "\n\n... (file truncated, total size > 50KB)", nil
			}

			return string(data), nil
		},
	})

	// Write File
	r.Register(ToolDef{
		Name:        "write_file",
		Risk:        RiskWrite,
		Description: "Writes or overwrites content to a specified file path",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":    map[string]interface{}{"type": "string", "description": "Path to write the file to"},
				"content": map[string]interface{}{"type": "string", "description": "Full file content"},
			},
			"required": []string{"path", "content"},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			path, _ := args["path"].(string)
			content, _ := args["content"].(string)
			if path == "" {
				return "", fmt.Errorf("missing 'path' argument")
			}

			dir := filepath.Dir(path)
			if dir != "" && dir != "." {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return "", fmt.Errorf("failed to create directory '%s': %w", dir, err)
				}
			}

			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return "", fmt.Errorf("failed to write file '%s': %w", path, err)
			}

			return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
		},
	})

	// Edit File (Find and Replace)
	r.Register(ToolDef{
		Name:        "edit_file",
		Risk:        RiskWrite,
		Description: "Replaces target content with replacement text in a file",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":        map[string]interface{}{"type": "string", "description": "Path to file to edit"},
				"target":      map[string]interface{}{"type": "string", "description": "Exact text to find and replace"},
				"replacement": map[string]interface{}{"type": "string", "description": "New replacement text"},
			},
			"required": []string{"path", "target", "replacement"},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			path, _ := args["path"].(string)
			target, _ := args["target"].(string)
			replacement, _ := args["replacement"].(string)
			if path == "" || target == "" {
				return "", fmt.Errorf("missing required 'path' or 'target' parameters")
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("failed to read file '%s': %w", path, err)
			}

			content := string(data)
			if !strings.Contains(content, target) {
				return "", fmt.Errorf("target string not found in '%s'", path)
			}

			newContent := strings.Replace(content, target, replacement, 1)
			if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
				return "", fmt.Errorf("failed to save edits to '%s': %w", path, err)
			}

			return fmt.Sprintf("Successfully replaced target text in %s", path), nil
		},
	})

	// List Directory
	r.Register(ToolDef{
		Name:        "list_dir",
		Description: "Lists files and subdirectories in a directory path",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "Directory path (default: current dir '.')"},
			},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			path, _ := args["path"].(string)
			if path == "" {
				path = "."
			}

			entries, err := os.ReadDir(path)
			if err != nil {
				return "", fmt.Errorf("failed to read directory '%s': %w", path, err)
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Directory listing for '%s':\n", path))
			for _, e := range entries {
				info, _ := e.Info()
				size := int64(0)
				if info != nil {
					size = info.Size()
				}
				typeStr := "FILE"
				if e.IsDir() {
					typeStr = "DIR "
				}
				sb.WriteString(fmt.Sprintf("  [%s] %-30s (%d bytes)\n", typeStr, e.Name(), size))
			}

			return sb.String(), nil
		},
	})

	// Search Code (Grep)
	r.Register(ToolDef{
		Name:        "find_files",
		Description: "Finds files whose names match a glob pattern (e.g. '*.go', 'config*.json') anywhere under a directory. Much faster than grep when you only need paths.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{"type": "string", "description": "Glob pattern or substring matched case-insensitively against file names"},
				"path":    map[string]interface{}{"type": "string", "description": "Root directory (default: '.')"},
				"limit":   map[string]interface{}{"type": "integer", "description": "Max results (default 200)"},
			},
			"required": []string{"pattern"},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			pattern, _ := args["pattern"].(string)
			if pattern == "" {
				pattern, _ = args["input"].(string)
			}
			if pattern == "" {
				return "", fmt.Errorf("missing 'pattern' argument")
			}
			root, _ := args["path"].(string)
			if root == "" {
				root = "."
			}
			limit := 200
			if v, ok := args["limit"].(float64); ok && v > 0 {
				limit = int(v)
			}

			hasGlob := strings.ContainsAny(pattern, "*?[")
			patLower := strings.ToLower(pattern)
			var found []string

			err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() {
					name := d.Name()
					if name == ".git" || name == "node_modules" || name == "vendor" || name == ".venv" || name == "__pycache__" || name == "dist" || name == "build" {
						return fs.SkipDir
					}
					return nil
				}
				name := d.Name()
				match := false
				if hasGlob {
					if ok, _ := filepath.Match(patLower, strings.ToLower(name)); ok {
						match = true
					}
				} else if strings.Contains(strings.ToLower(name), patLower) {
					match = true
				}
				if match {
					found = append(found, path)
					if len(found) >= limit {
						return fs.SkipAll
					}
				}
				return nil
			})
			if err != nil {
				return "", fmt.Errorf("error during find: %w", err)
			}
			if len(found) == 0 {
				return fmt.Sprintf("No files matching '%s' under %s", pattern, root), nil
			}
			return fmt.Sprintf("Found %d file(s) matching '%s':\n%s", len(found), pattern, strings.Join(found, "\n")), nil
		},
	})

	// Search Code (Grep)
	r.Register(ToolDef{
		Name:        "search_code",
		Description: "Searches for a text pattern across all project files",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string", "description": "Text pattern to search for"},
				"path":  map[string]interface{}{"type": "string", "description": "Root directory (default: '.')"},
			},
			"required": []string{"query"},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			query, _ := args["query"].(string)
			if query == "" {
				query, _ = args["input"].(string)
			}
			if query == "" {
				return "", fmt.Errorf("missing 'query' parameter")
			}

			root, _ := args["path"].(string)
			if root == "" {
				root = "."
			}

			var matches []string
			queryLower := strings.ToLower(query)

			err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() {
					name := d.Name()
					if name == ".git" || name == "node_modules" || name == "vendor" || name == ".gemini" {
						return filepath.SkipDir
					}
					return nil
				}

				// Skip binary or huge files
				if info, _ := d.Info(); info != nil && info.Size() > 500000 {
					return nil
				}

				file, err := os.Open(path)
				if err != nil {
					return nil
				}
				defer file.Close()

				scanner := bufio.NewScanner(file)
				lineNum := 1
				for scanner.Scan() {
					line := scanner.Text()
					if strings.Contains(strings.ToLower(line), queryLower) {
						matches = append(matches, fmt.Sprintf("%s:%d: %s", path, lineNum, strings.TrimSpace(line)))
						if len(matches) >= 50 {
							return filepath.SkipAll
						}
					}
					lineNum++
				}
				return nil
			})

			if err != nil {
				return "", fmt.Errorf("error during search: %w", err)
			}

			if len(matches) == 0 {
				return fmt.Sprintf("No matches found for pattern: '%s'", query), nil
			}

			return fmt.Sprintf("Found %d matches:\n%s", len(matches), strings.Join(matches, "\n")), nil
		},
	})
}

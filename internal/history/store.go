package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"shellsage/internal/branch"
	"shellsage/internal/config"
)

// ConversationMetadata contains summary stats for conversation files
type ConversationMetadata struct {
	Filename     string    `json:"filename"`
	Title        string    `json:"title"`
	PersonaID    string    `json:"persona_id"`
	MessageCount int       `json:"message_count"`
	TotalTokens  int       `json:"total_tokens"`
	UpdatedAt    time.Time `json:"updated_at"`
	Path         string    `json:"path"`
}

// GetConversationsDir returns the canonical directory for saved conversations:
// ~/.shellsage/conversations (stable across projects, unlike the old CWD-based
// ./conversations which silently scattered sessions).
func GetConversationsDir() string {
	dir := filepath.Join(config.GetUserConfigDir(), "conversations")
	_ = os.MkdirAll(dir, 0700)
	return dir
}

// legacyConversationsDir keeps pre-v4 sessions discoverable.
func legacyConversationsDir() string { return "conversations" }

// SaveTree saves a conversation tree to JSON (owner-only readable: transcripts
// can contain proprietary code).
func SaveTree(tree *branch.ConversationTree, filename string) (string, error) {
	if filename == "" {
		filename = fmt.Sprintf("chat_%s.json", time.Now().Format("2006-01-02_15-04-05"))
	}
	if !strings.HasSuffix(filename, ".json") {
		filename += ".json"
	}
	// Sanitize to a single path component.
	filename = filepath.Base(filename)

	saveDir := GetConversationsDir()
	path := filepath.Join(saveDir, filename)

	data, err := tree.ToJSON()
	if err != nil {
		return "", fmt.Errorf("failed to serialize conversation tree: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", fmt.Errorf("failed to save conversation to %s: %w", path, err)
	}
	return path, nil
}

// resolveConversationPath finds a saved conversation by name in the home dir
// or the legacy CWD dir (absolute paths pass through).
func resolveConversationPath(filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}
	base := filepath.Base(filename)
	candidate := filepath.Join(GetConversationsDir(), base)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	legacy := filepath.Join(legacyConversationsDir(), base)
	if _, err := os.Stat(legacy); err == nil {
		return legacy
	}
	return candidate
}

// LoadTree loads a conversation tree from a file
func LoadTree(filename string) (*branch.ConversationTree, error) {
	path := resolveConversationPath(filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file '%s': %w", path, err)
	}

	// Try loading as new Tree format
	tree, err := branch.FromJSON(data)
	if err == nil && tree.Nodes != nil && len(tree.Nodes) > 0 {
		if tree.Title == "" || tree.Title == "New Conversation" {
			tree.Title = strings.TrimSuffix(filepath.Base(path), ".json")
		}
		return tree, nil
	}

	// Fallback to legacy v1/v2 format
	var legacyData struct {
		Model    string `json:"model"`
		Persona  string `json:"persona"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	if err := json.Unmarshal(data, &legacyData); err == nil && len(legacyData.Messages) > 0 {
		legacyTree := branch.NewConversationTree(legacyData.Persona)
		for _, m := range legacyData.Messages {
			legacyTree.AddMessage(m.Role, m.Content, legacyData.Model, branch.TreeNode{}.Usage)
		}
		return legacyTree, nil
	}

	return nil, fmt.Errorf("unrecognized conversation file format in '%s'", path)
}

// DeleteTree removes a saved conversation file.
func DeleteTree(filename string) (string, error) {
	path := resolveConversationPath(filename)
	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("failed to delete '%s': %w", path, err)
	}
	return path, nil
}

// RenameTree renames a saved conversation (and its embedded title when empty).
func RenameTree(oldName, newName string) (string, error) {
	if !strings.HasSuffix(newName, ".json") {
		newName += ".json"
	}
	src := resolveConversationPath(oldName)
	dst := filepath.Join(GetConversationsDir(), filepath.Base(newName))
	if src == dst {
		return dst, nil
	}
	if _, err := os.Stat(dst); err == nil {
		return "", fmt.Errorf("target already exists: %s", dst)
	}
	if err := os.Rename(src, dst); err != nil {
		return "", fmt.Errorf("failed to rename: %w", err)
	}
	return dst, nil
}

// ListSavedConversations returns a list of metadata for all saved conversations
// (canonical dir first, legacy ./conversations merged, deduplicated by name).
func ListSavedConversations() []ConversationMetadata {
	seen := map[string]bool{}
	var list []ConversationMetadata

	collect := func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || seen[e.Name()] {
				continue
			}
			seen[e.Name()] = true
			path := filepath.Join(dir, e.Name())
			if tree, err := LoadTree(path); err == nil {
				pathNodes := tree.GetActivePath()
				totalTokens := 0
				for _, n := range pathNodes {
					totalTokens += n.Usage.TotalTokens
				}

				list = append(list, ConversationMetadata{
					Filename:     e.Name(),
					Title:        tree.Title,
					PersonaID:    tree.PersonaID,
					MessageCount: len(pathNodes),
					TotalTokens:  totalTokens,
					UpdatedAt:    tree.UpdatedAt,
					Path:         path,
				})
			}
		}
	}

	collect(GetConversationsDir())
	collect(legacyConversationsDir())

	sort.Slice(list, func(i, j int) bool {
		return list[i].UpdatedAt.After(list[j].UpdatedAt)
	})

	return list
}

// LatestConversation returns the most recently updated saved file name (for /resume).
func LatestConversation() (string, bool) {
	list := ListSavedConversations()
	if len(list) == 0 {
		return "", false
	}
	return list[0].Filename, true
}

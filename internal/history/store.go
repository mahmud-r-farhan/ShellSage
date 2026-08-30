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
)

// ConversationMetadata contains summary stats for conversation files
type ConversationMetadata struct {
	Filename     string    `json:"filename"`
	Title        string    `json:"title"`
	PersonaID    string    `json:"persona_id"`
	MessageCount int       `json:"message_count"`
	TotalTokens  int       `json:"total_tokens"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// GetConversationsDir returns the directory for saved conversation JSON files
func GetConversationsDir() string {
	// First check local conversations dir, fallback to user home config
	dir := "conversations"
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// SaveTree saves a conversation tree to JSON
func SaveTree(tree *branch.ConversationTree, filename string) (string, error) {
	if filename == "" {
		filename = fmt.Sprintf("chat_%s.json", time.Now().Format("2006-01-02_15-04-05"))
	}
	if !strings.HasSuffix(filename, ".json") {
		filename += ".json"
	}

	saveDir := GetConversationsDir()
	path := filepath.Join(saveDir, filepath.Base(filename))

	data, err := tree.ToJSON()
	if err != nil {
		return "", fmt.Errorf("failed to serialize conversation tree: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save conversation to %s: %w", path, err)
	}

	return path, nil
}

// LoadTree loads a conversation tree from a file
func LoadTree(filename string) (*branch.ConversationTree, error) {
	if !filepath.IsAbs(filename) {
		filename = filepath.Join(GetConversationsDir(), filepath.Base(filename))
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file '%s': %w", filename, err)
	}

	// Try loading as new Tree format
	tree, err := branch.FromJSON(data)
	if err == nil && tree.Nodes != nil && len(tree.Nodes) > 0 {
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

	return nil, fmt.Errorf("unrecognized conversation file format in '%s'", filename)
}

// ListSavedConversations returns a list of metadata for all saved conversations
func ListSavedConversations() []ConversationMetadata {
	dir := GetConversationsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var list []ConversationMetadata
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
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
				})
			}
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].UpdatedAt.After(list[j].UpdatedAt)
	})

	return list
}

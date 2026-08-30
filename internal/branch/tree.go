package branch

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"shellsage/internal/provider"
)

var nodeSeq uint64

// TreeNode represents a single message in the conversation tree
type TreeNode struct {
	ID        string              `json:"id"`
	ParentID  string              `json:"parent_id,omitempty"`
	Role      string              `json:"role"`
	Content   string              `json:"content"`
	Timestamp time.Time           `json:"timestamp"`
	Model     string              `json:"model,omitempty"`
	Usage     provider.TokenUsage `json:"usage,omitempty"`
	Children  []string            `json:"children,omitempty"`
}

// ConversationTree manages message branching and multi-path history
type ConversationTree struct {
	Nodes        map[string]*TreeNode `json:"nodes"`
	RootNodeIDs  []string             `json:"root_node_ids"`
	ActiveNodeID string               `json:"active_node_id"`
	Title        string               `json:"title"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
	PersonaID    string               `json:"persona_id"`
}

// NewConversationTree initializes a new conversation tree
func NewConversationTree(personaID string) *ConversationTree {
	return &ConversationTree{
		Nodes:       make(map[string]*TreeNode),
		RootNodeIDs: make([]string, 0),
		PersonaID:   personaID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Title:       "New Conversation",
	}
}

// AddMessage appends a message as a child of the current active node
func (t *ConversationTree) AddMessage(role, content, model string, usage provider.TokenUsage) *TreeNode {
	seq := atomic.AddUint64(&nodeSeq, 1)
	id := fmt.Sprintf("msg_%d_%d_%s", time.Now().UnixNano(), seq, role[:1])
	node := &TreeNode{
		ID:        id,
		ParentID:  t.ActiveNodeID,
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Model:     model,
		Usage:     usage,
		Children:  make([]string, 0),
	}

	t.Nodes[id] = node
	t.UpdatedAt = time.Now()

	if t.ActiveNodeID == "" {
		t.RootNodeIDs = append(t.RootNodeIDs, id)
	} else if parent, ok := t.Nodes[t.ActiveNodeID]; ok {
		parent.Children = append(parent.Children, id)
	}

	t.ActiveNodeID = id

	// Auto-title from first user message if default title
	if role == "user" && t.Title == "New Conversation" {
		snippet := strings.TrimSpace(content)
		if len(snippet) > 40 {
			snippet = snippet[:40] + "..."
		}
		t.Title = snippet
	}

	return node
}

// GetActivePath returns the slice of TreeNodes from root to the active node
func (t *ConversationTree) GetActivePath() []*TreeNode {
	if t.ActiveNodeID == "" {
		return []*TreeNode{}
	}

	var path []*TreeNode
	currID := t.ActiveNodeID

	for currID != "" {
		node, ok := t.Nodes[currID]
		if !ok {
			break
		}
		path = append([]*TreeNode{node}, path...)
		currID = node.ParentID
	}

	return path
}

// GetActiveMessages converts the active path to provider Messages for LLM context
func (t *ConversationTree) GetActiveMessages() []provider.Message {
	path := t.GetActivePath()
	messages := make([]provider.Message, 0, len(path))
	for _, n := range path {
		messages = append(messages, provider.Message{
			Role:    n.Role,
			Content: n.Content,
		})
	}
	return messages
}

// GetLastMessage returns the active leaf node
func (t *ConversationTree) GetLastMessage() *TreeNode {
	if t.ActiveNodeID == "" {
		return nil
	}
	return t.Nodes[t.ActiveNodeID]
}

// RewindLastUserTurn moves the active pointer back to before the last user/assistant exchange
// enabling an alternative response generation
func (t *ConversationTree) RewindForAlternative() (*TreeNode, bool) {
	last := t.GetLastMessage()
	if last == nil {
		return nil, false
	}

	// If last is assistant, rewind to parent (the user turn)
	if last.Role == "assistant" && last.ParentID != "" {
		userNode := t.Nodes[last.ParentID]
		if userNode != nil {
			t.ActiveNodeID = userNode.ID
			return userNode, true
		}
	}

	return nil, false
}

// Branch represents an identifiable path through the conversation
type BranchInfo struct {
	Index       int
	LeafNodeID  string
	MessageCount int
	LastSnippet string
	IsActive    bool
}

// ListBranches identifies all leaf nodes (endpoints) representing distinct conversation branches
func (t *ConversationTree) ListBranches() []BranchInfo {
	var leaves []*TreeNode
	for _, node := range t.Nodes {
		if len(node.Children) == 0 {
			leaves = append(leaves, node)
		}
	}

	var branches []BranchInfo
	for i, leaf := range leaves {
		// Calculate message count along path
		count := 0
		curr := leaf
		for curr != nil {
			count++
			curr = t.Nodes[curr.ParentID]
		}

		snippet := strings.ReplaceAll(leaf.Content, "\n", " ")
		if len(snippet) > 60 {
			snippet = snippet[:60] + "..."
		}

		branches = append(branches, BranchInfo{
			Index:        i + 1,
			LeafNodeID:   leaf.ID,
			MessageCount: count,
			LastSnippet:  snippet,
			IsActive:     leaf.ID == t.ActiveNodeID,
		})
	}

	return branches
}

// SwitchBranch activates a specific leaf branch
func (t *ConversationTree) SwitchBranch(leafID string) bool {
	if _, ok := t.Nodes[leafID]; ok {
		t.ActiveNodeID = leafID
		return true
	}
	return false
}

// RenderTreeAscii generates a visual ASCII map of the conversation tree
func (t *ConversationTree) RenderTreeAscii() string {
	if len(t.RootNodeIDs) == 0 {
		return "(Empty conversation tree)"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🌳 Conversation Tree: %s\n", t.Title))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	for _, rootID := range t.RootNodeIDs {
		t.renderNodeAscii(&sb, rootID, "", true)
	}

	return sb.String()
}

func (t *ConversationTree) renderNodeAscii(sb *strings.Builder, nodeID, prefix string, isTail bool) {
	node, ok := t.Nodes[nodeID]
	if !ok {
		return
	}

	connector := "├── "
	if isTail {
		connector = "└── "
	}

	activeMarker := ""
	if node.ID == t.ActiveNodeID {
		activeMarker = " [ACTIVE]"
	}

	icon := "👤"
	if node.Role == "assistant" {
		icon = "🤖"
	} else if node.Role == "system" {
		icon = "⚙️"
	}

	snippet := strings.ReplaceAll(node.Content, "\n", " ")
	if len(snippet) > 45 {
		snippet = snippet[:45] + "..."
	}

	sb.WriteString(fmt.Sprintf("%s%s%s %s: \"%s\"%s\n", prefix, connector, icon, strings.ToUpper(node.Role), snippet, activeMarker))

	childPrefix := prefix
	if isTail {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	for i, childID := range node.Children {
		isChildTail := i == len(node.Children)-1
		t.renderNodeAscii(sb, childID, childPrefix, isChildTail)
	}
}

// ToJSON serializes the tree
func (t *ConversationTree) ToJSON() ([]byte, error) {
	return json.MarshalIndent(t, "", "  ")
}

// FromJSON loads a conversation tree
func FromJSON(data []byte) (*ConversationTree, error) {
	var tree ConversationTree
	if err := json.Unmarshal(data, &tree); err != nil {
		return nil, err
	}
	return &tree, nil
}

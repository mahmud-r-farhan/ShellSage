package branch

import (
	"testing"

	"shellsage/internal/provider"
)

func TestConversationTreeBranching(t *testing.T) {
	tree := NewConversationTree("developer")

	// Add 1st user message
	n1 := tree.AddMessage("user", "Write a hello world in Go", "model1", provider.TokenUsage{TotalTokens: 10})
	// Add 1st assistant response
	_ = tree.AddMessage("assistant", "fmt.Println(\"Hello World\")", "model1", provider.TokenUsage{TotalTokens: 20})

	// Check path length
	path := tree.GetActivePath()
	if len(path) != 2 {
		t.Fatalf("expected path length 2, got %d", len(path))
	}

	// Rewind for alternative response
	userNode, ok := tree.RewindForAlternative()
	if !ok || userNode.ID != n1.ID {
		t.Fatalf("failed to rewind for alternative response")
	}

	// Add 2nd assistant alternative response
	n3 := tree.AddMessage("assistant", "package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"Hello, World!\") }", "model1", provider.TokenUsage{TotalTokens: 30})

	// Now parent should have 2 children
	if len(n1.Children) != 2 {
		t.Fatalf("expected 2 children on user node, got %d", len(n1.Children))
	}

	// Verify branches
	branches := tree.ListBranches()
	if len(branches) != 2 {
		t.Fatalf("expected 2 branches, got %d", len(branches))
	}

	// Active node should be n3
	if tree.ActiveNodeID != n3.ID {
		t.Errorf("expected active node %s, got %s", n3.ID, tree.ActiveNodeID)
	}

	// Visual ASCII render shouldn't panic
	ascii := tree.RenderTreeAscii()
	if len(ascii) == 0 {
		t.Errorf("empty ASCII render")
	}
}

package history

import (
	"testing"

	"shellsage/internal/branch"
	"shellsage/internal/provider"
)

func TestSessionAnalytics(t *testing.T) {
	analytics := NewSessionAnalytics()
	analytics.RecordTurn("gpt-4o", 1000, 500)
	analytics.RecordTurn("gpt-4o-mini", 2000, 1000)

	if analytics.TotalPromptTokens != 3000 {
		t.Errorf("expected 3000 prompt tokens, got %d", analytics.TotalPromptTokens)
	}
	if analytics.TotalOutputTokens != 1500 {
		t.Errorf("expected 1500 output tokens, got %d", analytics.TotalOutputTokens)
	}
	if analytics.TotalTokens != 4500 {
		t.Errorf("expected 4500 total tokens, got %d", analytics.TotalTokens)
	}
	if analytics.EstimatedTotalCost <= 0 {
		t.Errorf("expected non-zero estimated cost")
	}

	tree := branch.NewConversationTree("developer")
	tree.AddMessage("user", "Hello", "gpt-4o", provider.TokenUsage{TotalTokens: 10})
	summary := analytics.FormatSummary("Developer", "gpt-4o", tree)
	if len(summary) == 0 {
		t.Errorf("empty summary string")
	}
}

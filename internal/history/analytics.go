package history

import (
	"fmt"
	"strings"
	"time"

	"shellsage/internal/branch"
)

// ModelPricing holds estimated cost per 1 Million tokens (input/output)
type ModelPricing struct {
	InputPer1M  float64
	OutputPer1M float64
}

// Estimated pricing matrix for standard models
var PricingTable = map[string]ModelPricing{
	"gpt-4o":                   {InputPer1M: 2.50, OutputPer1M: 10.00},
	"gpt-4o-mini":              {InputPer1M: 0.15, OutputPer1M: 0.60},
	"claude-3-5-sonnet-latest": {InputPer1M: 3.00, OutputPer1M: 15.00},
	"claude-3-5-haiku-latest":  {InputPer1M: 0.80, OutputPer1M: 4.00},
	"gemini-2.0-flash":         {InputPer1M: 0.10, OutputPer1M: 0.40},
	"deepseek-chat":            {InputPer1M: 0.14, OutputPer1M: 0.28},
	"llama-3.3-70b-versatile":  {InputPer1M: 0.59, OutputPer1M: 0.79},
	"openrouter/free":          {InputPer1M: 0.00, OutputPer1M: 0.00},
}

// SessionAnalytics tracks aggregate session usage
type SessionAnalytics struct {
	StartTime          time.Time
	TotalPromptTokens  int
	TotalOutputTokens  int
	TotalTokens        int
	TotalTurns         int
	ModelUsage         map[string]int
	EstimatedTotalCost float64
}

// NewSessionAnalytics initializes session analytics
func NewSessionAnalytics() *SessionAnalytics {
	return &SessionAnalytics{
		StartTime:  time.Now(),
		ModelUsage: make(map[string]int),
	}
}

// RecordTurn logs token usage from a completed message turn
func (s *SessionAnalytics) RecordTurn(model string, promptTokens, completionTokens int) {
	s.TotalPromptTokens += promptTokens
	s.TotalOutputTokens += completionTokens
	s.TotalTokens += (promptTokens + completionTokens)
	s.TotalTurns++

	if model != "" {
		s.ModelUsage[model] += (promptTokens + completionTokens)
	}

	// Calculate cost
	pricing, ok := PricingTable[model]
	if !ok {
		// Fallback average estimate: $1.00 / 1M tokens
		pricing = ModelPricing{InputPer1M: 1.00, OutputPer1M: 2.00}
	}

	turnCost := (float64(promptTokens)/1000000.0)*pricing.InputPer1M + (float64(completionTokens)/1000000.0)*pricing.OutputPer1M
	s.EstimatedTotalCost += turnCost
}

// FormatSummary formats analytics into a visual terminal string
func (s *SessionAnalytics) FormatSummary(currentPersona, currentModel string, tree *branch.ConversationTree) string {
	duration := time.Since(s.StartTime).Round(time.Second)

	var sb strings.Builder
	sb.WriteString("\n📊 ━━━ ShellSage Session Statistics & Analytics ━━━\n")
	sb.WriteString(fmt.Sprintf("  ⏱️  Session Duration:    %v\n", duration))
	sb.WriteString(fmt.Sprintf("  💬  Total Message Turns: %d\n", s.TotalTurns))
	sb.WriteString(fmt.Sprintf("  🎭  Active Persona:      %s\n", currentPersona))
	sb.WriteString(fmt.Sprintf("  🤖  Active Model:        %s\n", currentModel))
	sb.WriteString("───────────────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("  📥  Prompt Tokens:       %d\n", s.TotalPromptTokens))
	sb.WriteString(fmt.Sprintf("  📤  Completion Tokens:   %d\n", s.TotalOutputTokens))
	sb.WriteString(fmt.Sprintf("  🔢  Total Tokens:        %d\n", s.TotalTokens))
	sb.WriteString(fmt.Sprintf("  💰  Estimated API Cost:  $%.4f USD\n", s.EstimatedTotalCost))
	sb.WriteString("───────────────────────────────────────────────────\n")

	if len(s.ModelUsage) > 0 {
		sb.WriteString("  📈  Model Usage Breakdown:\n")
		for m, tok := range s.ModelUsage {
			pct := float64(tok) / float64(s.TotalTokens) * 100.0
			sb.WriteString(fmt.Sprintf("      • %-26s : %6d tokens (%4.1f%%)\n", m, tok, pct))
		}
	}

	if tree != nil {
		branches := tree.ListBranches()
		sb.WriteString(fmt.Sprintf("  🌿  Active Chat Branches: %d\n", len(branches)))
	}

	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	return sb.String()
}

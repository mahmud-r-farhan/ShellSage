package agent

import (
	"context"
	"fmt"

	"shellsage/internal/provider"
)

// GeneratePlan produces a structured architecture & implementation plan
func (a *Agent) GeneratePlan(ctx context.Context, requirement string) (string, error) {
	systemPrompt := `You are an Enterprise Software Architect and Agile Project Lead.
Create a comprehensive, production-grade Implementation Plan for the requested feature or system.

Format your plan with the following structure:
# [Feature/Goal Title]

## 1. Problem Statement & Objectives
- High-level overview and success criteria

## 2. Architecture & Design Decisions
- Component breakdown and tech stack considerations
- Potential trade-offs

## 3. Step-by-Step Implementation Tasks
- Grouped into chronological milestones with checkboxes
- Concrete file paths and function signatures

## 4. Edge Cases & Risk Mitigation
- Potential failure modes, race conditions, edge cases

## 5. Verification & Testing Strategy
- Automated unit/integration tests and manual verification steps
`

	req := &provider.ChatRequest{
		Model: a.model,
		Messages: []provider.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("Please generate a detailed implementation plan for: %s", requirement)},
		},
		Temperature: 0.5,
	}

	resp, err := a.provider.Chat(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

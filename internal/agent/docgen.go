package agent

import (
	"context"
	"fmt"

	"shellsage/internal/provider"
)

// GenerateDocumentation generates markdown documentation or docstrings for code files
func (a *Agent) GenerateDocumentation(ctx context.Context, codeOrPath, codeContent string) (string, error) {
	systemPrompt := `You are an expert Technical Writer and Developer Advocate.
Generate clear, comprehensive, and beautiful documentation in GitHub-Flavored Markdown for the provided code.

Include:
- 📖 Overview & Architecture Role
- 🚀 Quick Start / Usage Examples
- ⚙️ API / Struct / Function Reference with parameters & return values
- 💡 Key Edge Cases & Error Handling
`

	prompt := fmt.Sprintf("Generate documentation for file `%s`:\n\n```\n%s\n```", codeOrPath, codeContent)
	if codeContent == "" {
		prompt = fmt.Sprintf("Generate documentation for: %s", codeOrPath)
	}

	req := &provider.ChatRequest{
		Model: a.model,
		Messages: []provider.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.4,
	}

	resp, err := a.provider.Chat(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

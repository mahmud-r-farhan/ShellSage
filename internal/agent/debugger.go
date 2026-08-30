package agent

import (
	"context"
	"fmt"

	"shellsage/internal/provider"
)

// DiagnoseError performs root cause analysis on error logs, stack traces, or failing code
func (a *Agent) DiagnoseError(ctx context.Context, errorDetails string) (string, error) {
	systemPrompt := `You are a Principal Debugging Engineer and Systems Diagnostician.
Given an error message, stack trace, or failing test output, perform a deep root-cause analysis.

Format your response as follows:
### 🔍 Root Cause Analysis
Explain clearly WHY this error occurred and the underlying mechanism.

### 🛠️ Step-by-Step Fix
Provide the exact code modification or configuration update needed.

### 🛡️ Regression Prevention
Recommend best practices, defensive coding patterns, or unit tests to prevent this from happening again.
`

	req := &provider.ChatRequest{
		Model: a.model,
		Messages: []provider.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("Analyze and diagnose this issue:\n\n%s", errorDetails)},
		},
		Temperature: 0.3,
	}

	resp, err := a.provider.Chat(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

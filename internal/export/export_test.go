package export

import (
	"strings"
	"testing"

	"shellsage/internal/branch"
	"shellsage/internal/provider"
)

func TestExportMarkdownAndHTML(t *testing.T) {
	tree := branch.NewConversationTree("developer")
	tree.Title = "Test Go Optimization Chat"
	tree.AddMessage("user", "How do I optimize loops in Go?", "gpt-4o", provider.TokenUsage{PromptTokens: 10, CompletionTokens: 0, TotalTokens: 10})
	tree.AddMessage("assistant", "Use bounds check elimination.", "gpt-4o", provider.TokenUsage{PromptTokens: 0, CompletionTokens: 15, TotalTokens: 15})

	md := ExportToMarkdown(tree)
	if !strings.Contains(md, "# 💬 Test Go Optimization Chat") {
		t.Errorf("markdown missing title")
	}
	if !strings.Contains(md, "Use bounds check elimination") {
		t.Errorf("markdown missing content")
	}

	html := ExportToHTML(tree)
	if !strings.Contains(html, "<title>Test Go Optimization Chat - ShellSage Export</title>") {
		t.Errorf("html missing title")
	}

	pdfBytes, err := ExportToPDF(tree)
	if err != nil {
		t.Fatalf("failed to generate PDF: %v", err)
	}
	if !strings.HasPrefix(string(pdfBytes), "%PDF-1.4") {
		t.Errorf("PDF header missing")
	}
}

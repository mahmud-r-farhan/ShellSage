package clipboard

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/atotto/clipboard"
)

// CopyText copies raw text to the OS clipboard
func CopyText(text string) error {
	if text == "" {
		return fmt.Errorf("nothing to copy (text is empty)")
	}
	return clipboard.WriteAll(text)
}

// ExtractCodeBlocks parses Markdown content and extracts only the code inside code fences
func ExtractCodeBlocks(markdown string) string {
	re := regexp.MustCompile("(?s)```(?:[a-zA-Z0-9_-]+)?\\s*\n(.*?)\\s*```")
	matches := re.FindAllStringSubmatch(markdown, -1)

	if len(matches) == 0 {
		return markdown
	}

	var codeSnippets []string
	for _, m := range matches {
		if len(m) > 1 {
			codeSnippets = append(codeSnippets, strings.TrimSpace(m[1]))
		}
	}

	return strings.Join(codeSnippets, "\n\n// ── Next Snippet ──\n\n")
}

// CopyCodeOnly extracts all code blocks and writes them to the clipboard
func CopyCodeOnly(markdown string) (string, error) {
	code := ExtractCodeBlocks(markdown)
	if err := CopyText(code); err != nil {
		return "", err
	}
	return code, nil
}

package clipboard

import (
	"strings"
	"testing"
)

func TestExtractCodeBlocks(t *testing.T) {
	md := `Here is some explanation:
` + "```go\npackage main\n\nfunc main() {}\n```" + `
And another snippet:
` + "```python\nprint('hello')\n```"

	extracted := ExtractCodeBlocks(md)
	if !strings.Contains(extracted, "package main") || !strings.Contains(extracted, "print('hello')") {
		t.Errorf("failed to extract code blocks: %s", extracted)
	}
	if strings.Contains(extracted, "Here is some explanation") {
		t.Errorf("extracted code should not contain surrounding markdown prose")
	}
}

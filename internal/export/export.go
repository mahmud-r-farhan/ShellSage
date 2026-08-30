package export

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"shellsage/internal/branch"
)

// ExportFormat specifies the output format
type ExportFormat string

const (
	FormatMarkdown ExportFormat = "md"
	FormatPDF      ExportFormat = "pdf"
	FormatHTML     ExportFormat = "html"
	FormatJSON     ExportFormat = "json"
)

// GetDefaultExportDir returns the directory where exported chats are saved
func GetDefaultExportDir() string {
	dir := "exports"
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// ExportConversation writes the conversation tree active path or full tree in the requested format
func ExportConversation(tree *branch.ConversationTree, format ExportFormat, filename string) (string, error) {
	if filename == "" {
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		filename = fmt.Sprintf("chat_%s.%s", timestamp, string(format))
	}

	if !filepath.IsAbs(filename) {
		filename = filepath.Join(GetDefaultExportDir(), filename)
	}

	var data []byte
	var err error

	switch format {
	case FormatMarkdown, "markdown":
		data = []byte(ExportToMarkdown(tree))
	case FormatHTML:
		data = []byte(ExportToHTML(tree))
	case FormatJSON:
		data, err = tree.ToJSON()
	case FormatPDF:
		data, err = ExportToPDF(tree)
	default:
		data = []byte(ExportToMarkdown(tree))
	}

	if err != nil {
		return "", fmt.Errorf("failed to generate %s export: %w", format, err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write export file: %w", err)
	}

	return filename, nil
}

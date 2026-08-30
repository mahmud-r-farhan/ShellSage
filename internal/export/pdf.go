package export

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"shellsage/internal/branch"
)

// ExportToPDF generates a clean, valid PDF document in pure Go
func ExportToPDF(tree *branch.ConversationTree) ([]byte, error) {
	path := tree.GetActivePath()

	// Prepare text lines for PDF pages
	var lines []string
	lines = append(lines, fmt.Sprintf("ShellSage Chat Transcript: %s", sanitizePDFText(tree.Title)))
	lines = append(lines, fmt.Sprintf("Date: %s  |  Persona: %s", time.Now().Format("2006-01-02 15:04:05"), tree.PersonaID))
	lines = append(lines, "--------------------------------------------------------------------------------")
	lines = append(lines, "")

	for i, node := range path {
		role := "USER"
		if node.Role == "assistant" {
			role = "ASSISTANT"
			if node.Model != "" {
				role += fmt.Sprintf(" (%s)", node.Model)
			}
		} else if node.Role == "system" {
			role = "SYSTEM"
		}

		lines = append(lines, fmt.Sprintf("[%d] %s - %s", i+1, role, node.Timestamp.Format("15:04:05")))

		// Wrap long lines
		wrapped := wrapText(sanitizePDFText(node.Content), 85)
		for _, wLine := range wrapped {
			lines = append(lines, "    "+wLine)
		}
		lines = append(lines, "")
	}

	// Build PDF objects
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	type pdfObj struct {
		offset int
	}
	var offsets []int

	// 1 0 obj: Catalog
	offsets = append(offsets, buf.Len())
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// Split lines into pages (approx 45 lines per page)
	linesPerPage := 45
	var pageStreams [][]string
	for i := 0; i < len(lines); i += linesPerPage {
		end := i + linesPerPage
		if end > len(lines) {
			end = len(lines)
		}
		pageStreams = append(pageStreams, lines[i:end])
	}
	if len(pageStreams) == 0 {
		pageStreams = append(pageStreams, []string{"(No messages)"})
	}

	numPages := len(pageStreams)

	// Build page object references
	var pageRefStrings []string
	for i := 0; i < numPages; i++ {
		pageObjNum := 3 + i*2
		pageRefStrings = append(pageRefStrings, fmt.Sprintf("%d 0 R", pageObjNum))
	}

	// 2 0 obj: Pages
	offsets = append(offsets, buf.Len())
	buf.WriteString(fmt.Sprintf("2 0 obj\n<< /Type /Pages /Kids [ %s ] /Count %d >>\nendobj\n", strings.Join(pageRefStrings, " "), numPages))

	for pageIdx, pLines := range pageStreams {
		pageObjNum := 3 + pageIdx*2
		contentObjNum := pageObjNum + 1

		// Page object
		offsets = append(offsets, buf.Len())
		buf.WriteString(fmt.Sprintf("%d 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [ 0 0 612 792 ] /Contents %d 0 R /Resources << /Font << /F1 << /Type /Font /Subtype /Type1 /BaseFont /Courier >> >> >> >>\nendobj\n", pageObjNum, contentObjNum))

		// Stream content
		var streamBuf bytes.Buffer
		streamBuf.WriteString("BT\n/F1 9 Tf\n12 TL\n40 750 Td\n")
		for _, l := range pLines {
			streamBuf.WriteString(fmt.Sprintf("(%s) '\n", escapePDFString(l)))
		}
		streamBuf.WriteString("ET\n")

		streamBytes := streamBuf.Bytes()

		// Content stream object
		offsets = append(offsets, buf.Len())
		buf.WriteString(fmt.Sprintf("%d 0 obj\n<< /Length %d >>\nstream\n%sendstream\nendobj\n", contentObjNum, len(streamBytes), string(streamBytes)))
	}

	// Cross-Reference Table
	xrefOffset := buf.Len()
	buf.WriteString(fmt.Sprintf("xref\n0 %d\n0000000000 65535 f \n", len(offsets)+1))
	for _, off := range offsets {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", off))
	}

	// Trailer
	buf.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets)+1, xrefOffset))

	return buf.Bytes(), nil
}

func sanitizePDFText(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r >= 32 && r <= 126 {
			sb.WriteRune(r)
		} else if r == '\n' || r == '\t' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('?')
		}
	}
	return sb.String()
}

func escapePDFString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}

func wrapText(text string, maxLen int) []string {
	var result []string
	rawLines := strings.Split(text, "\n")

	for _, line := range rawLines {
		if len(line) <= maxLen {
			result = append(result, line)
			continue
		}

		words := strings.Fields(line)
		if len(words) == 0 {
			result = append(result, "")
			continue
		}

		var curr strings.Builder
		for _, w := range words {
			if curr.Len()+len(w)+1 > maxLen {
				if curr.Len() > 0 {
					result = append(result, curr.String())
					curr.Reset()
				}
			}
			if curr.Len() > 0 {
				curr.WriteString(" ")
			}
			curr.WriteString(w)
		}
		if curr.Len() > 0 {
			result = append(result, curr.String())
		}
	}

	return result
}

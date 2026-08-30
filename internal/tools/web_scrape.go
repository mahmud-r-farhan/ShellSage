package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func (r *ToolRegistry) registerWebScrapeTools() {
	r.Register(ToolDef{
		Name:        "web_scrape",
		Description: "Fetches and extracts clean readable text content from any public webpage/URL",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{"type": "string", "description": "URL to scrape (e.g. https://docs.github.com/...)"},
			},
			"required": []string{"url"},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			targetURL, _ := args["url"].(string)
			if targetURL == "" {
				targetURL, _ = args["input"].(string)
			}
			if targetURL == "" {
				return "", fmt.Errorf("missing 'url' parameter")
			}

			if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
				targetURL = "https://" + targetURL
			}

			client := &http.Client{Timeout: 20 * time.Second}
			req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
			if err != nil {
				return "", fmt.Errorf("invalid URL: %w", err)
			}
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
			req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

			resp, err := client.Do(req)
			if err != nil {
				return "", fmt.Errorf("failed to fetch URL '%s': %w", targetURL, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return "", fmt.Errorf("HTTP error %d fetching %s", resp.StatusCode, targetURL)
			}

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", fmt.Errorf("failed to read response: %w", err)
			}

			extracted := extractReadableText(string(bodyBytes))
			if len(extracted) > 15000 {
				extracted = extracted[:15000] + "\n\n... (webpage content truncated, total > 15KB)"
			}

			return fmt.Sprintf("=== Scraped content from %s ===\n\n%s", targetURL, extracted), nil
		},
	})
}

func extractReadableText(html string) string {
	// Strip script and style tags completely
	reScript := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	html = reScript.ReplaceAllString(html, "")

	reStyle := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	html = reStyle.ReplaceAllString(html, "")

	reNav := regexp.MustCompile(`(?is)<nav[^>]*>.*?</nav>`)
	html = reNav.ReplaceAllString(html, "")

	reFooter := regexp.MustCompile(`(?is)<footer[^>]*>.*?</footer>`)
	html = reFooter.ReplaceAllString(html, "")

	// Convert basic headers, paragraphs and breaks to newlines
	reBreak := regexp.MustCompile(`(?i)<(br|p|div|h1|h2|h3|h4|li)[^>]*>`)
	html = reBreak.ReplaceAllString(html, "\n")

	// Strip remaining HTML tags
	reTags := regexp.MustCompile(`<[^>]+>`)
	text := reTags.ReplaceAllString(html, " ")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	// Normalize whitespace
	lines := strings.Split(text, "\n")
	var cleanLines []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			cleanLines = append(cleanLines, trimmed)
		}
	}

	return strings.Join(cleanLines, "\n")
}

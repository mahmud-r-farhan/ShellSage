package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

func (r *ToolRegistry) registerWebSearchTools() {
	r.Register(ToolDef{
		Name:        "web_search",
		Description: "Searches the web for latest developer documentation, error solutions, and technical info",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string", "description": "Search keywords or question"},
			},
			"required": []string{"query"},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			query, _ := args["query"].(string)
			if query == "" {
				query, _ = args["input"].(string)
			}
			if query == "" {
				return "", fmt.Errorf("missing 'query' parameter")
			}

			client := &http.Client{Timeout: 15 * time.Second}

			// Try DuckDuckGo Instant Answer API first
			apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1", url.QueryEscape(query))
			req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
			if err != nil {
				return "", err
			}
			req.Header.Set("User-Agent", "ShellSage-CLI/3.0")

			resp, err := client.Do(req)
			var results strings.Builder

			if err == nil && resp.StatusCode == http.StatusOK {
				var ddg struct {
					AbstractText   string `json:"AbstractText"`
					AbstractSource string `json:"AbstractSource"`
					AbstractURL    string `json:"AbstractURL"`
					RelatedTopics  []struct {
						Text     string `json:"Text"`
						FirstURL string `json:"FirstURL"`
					} `json:"RelatedTopics"`
				}
				_ = json.NewDecoder(resp.Body).Decode(&ddg)
				resp.Body.Close()

				if ddg.AbstractText != "" {
					results.WriteString(fmt.Sprintf("Summary (%s):\n%s\nSource: %s\n\n", ddg.AbstractSource, ddg.AbstractText, ddg.AbstractURL))
				}

				if len(ddg.RelatedTopics) > 0 {
					results.WriteString("Key Results:\n")
					count := 0
					for _, topic := range ddg.RelatedTopics {
						if topic.Text != "" && topic.FirstURL != "" {
							count++
							results.WriteString(fmt.Sprintf("%d. %s\n   URL: %s\n", count, topic.Text, topic.FirstURL))
							if count >= 5 {
								break
							}
						}
					}
				}
			}

			// If DDG instant answer is insufficient, query HTML search
			if results.Len() < 50 {
				htmlSearchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))
				hReq, err := http.NewRequestWithContext(ctx, "GET", htmlSearchURL, nil)
				if err == nil {
					hReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
					hResp, err := client.Do(hReq)
					if err == nil && hResp.StatusCode == http.StatusOK {
						bodyBytes, _ := io.ReadAll(hResp.Body)
						hResp.Body.Close()

						parsedResults := parseDuckDuckGoHTML(string(bodyBytes))
						if len(parsedResults) > 0 {
							if results.Len() > 0 {
								results.WriteString("\nWeb Results:\n")
							} else {
								results.WriteString("Web Results for '" + query + "':\n\n")
							}
							for i, res := range parsedResults {
								results.WriteString(fmt.Sprintf("%d. %s\n   Snippet: %s\n   URL: %s\n\n", i+1, res.Title, res.Snippet, res.URL))
								if i >= 4 {
									break
								}
							}
						}
					}
				}
			}

			if results.Len() == 0 {
				return fmt.Sprintf("No web search results found for query '%s'. Try a broader keyword.", query), nil
			}

			return results.String(), nil
		},
	})
}

type searchItem struct {
	Title   string
	Snippet string
	URL     string
}

func parseDuckDuckGoHTML(html string) []searchItem {
	var items []searchItem

	// Match result snippets in DDG HTML
	titleRe := regexp.MustCompile(`<a class="result__url" href="([^"]+)">(?:<[^>]+>)*([^<]+)`)
	snippetRe := regexp.MustCompile(`<a class="result__snippet[^"]*"[^>]*>(.*?)</a>`)

	urls := titleRe.FindAllStringSubmatch(html, -1)
	snippets := snippetRe.FindAllStringSubmatch(html, -1)

	for i := 0; i < len(urls) && i < 5; i++ {
		cleanURL := strings.TrimSpace(urls[i][1])
		if strings.HasPrefix(cleanURL, "//") {
			cleanURL = "https:" + cleanURL
		}
		title := cleanHTML(urls[i][2])
		snippet := ""
		if i < len(snippets) {
			snippet = cleanHTML(snippets[i][1])
		}
		if title != "" && cleanURL != "" {
			items = append(items, searchItem{
				Title:   title,
				Snippet: snippet,
				URL:     cleanURL,
			})
		}
	}

	return items
}

func cleanHTML(s string) string {
	reTag := regexp.MustCompile(`<[^>]+>`)
	s = reTag.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	return strings.TrimSpace(s)
}

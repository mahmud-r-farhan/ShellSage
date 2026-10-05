package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// allowLocalNetTarget disables the SSRF guard for RFC1918/loopback targets —
// useful when testing local APIs. Link-local/metadata IPs stay blocked always.
func allowLocalNetTarget() bool {
	v := strings.ToLower(os.Getenv("SHELLSAGE_ALLOW_LOCAL_NET"))
	return v == "1" || v == "true" || v == "yes"
}

// blockedIPOrHost reports whether the resolved address is a link-local/metadata
// endpoint (always denied) or private space (denied unless SHELLSAGE_ALLOW_LOCAL_NET).
func blockedIPOrHost(host string) (string, bool) {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if ip := net.ParseIP(host); ip != nil {
		return inspectIP(ip)
	}
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", false // let the HTTP request fail naturally
	}
	for _, a := range addrs {
		if reason, blocked := inspectIP(net.ParseIP(a)); blocked {
			return reason, true
		}
	}
	return "", false
}

func inspectIP(ip net.IP) (string, bool) {
	if ip == nil {
		return "", false
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return "link-local address (cloud metadata ranges are blocked)", true
	}
	if ip.String() == "169.254.169.254" || ip.String() == "fd00:ec2::254" {
		return "cloud metadata service", true
	}
	if ip.IsUnspecified() {
		return "unspecified address", true
	}
	if (ip.IsPrivate() || ip.IsLoopback()) && !allowLocalNetTarget() {
		return "private/loopback network (set SHELLSAGE_ALLOW_LOCAL_NET=1 to allow for local API testing)", true
	}
	return "", false
}

// registerNetTools adds a generic HTTP client tool — debugging REST APIs,
// webhooks and health endpoints is one of the most common agent needs.
func (r *ToolRegistry) registerNetTools() {
	r.Register(ToolDef{
		Name:        "http_request",
		Description: "Sends an HTTP request (GET/POST/PUT/PATCH/DELETE) and returns status, selected response headers and body. GET is read-only; mutating methods require approval. Private/loopback hosts are blocked unless SHELLSAGE_ALLOW_LOCAL_NET=1; cloud metadata endpoints are always blocked.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url":     map[string]interface{}{"type": "string", "description": "Full URL including scheme"},
				"method":  map[string]interface{}{"type": "string", "description": "HTTP method (default GET)"},
				"headers": map[string]interface{}{"type": "object", "description": "Optional request headers, e.g. {\"Content-Type\":\"application/json\"}"},
				"body":    map[string]interface{}{"type": "string", "description": "Optional raw request body (JSON string, form data, ...)"},
			},
			"required": []string{"url"},
		},
		Risk: RiskNet,
		DynamicRisk: func(args map[string]interface{}) Risk {
			method, _ := args["method"].(string)
			if method == "" || strings.ToUpper(method) == "GET" || strings.ToUpper(method) == "HEAD" {
				return RiskRead
			}
			return RiskNet
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			targetURL, _ := args["url"].(string)
			if targetURL == "" {
				targetURL, _ = args["input"].(string)
			}
			if targetURL == "" {
				return "", fmt.Errorf("missing 'url' argument")
			}
			if !strings.Contains(targetURL, "://") {
				targetURL = "https://" + targetURL
			}

			method := "GET"
			if m, _ := args["method"].(string); m != "" {
				method = strings.ToUpper(m)
			}
			switch method {
			case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
			default:
				return "", fmt.Errorf("unsupported method %q", method)
			}

			// Parse host for SSRF guard
			host := targetURL
			if i := strings.Index(host, "://"); i >= 0 {
				host = host[i+3:]
			}
			if i := strings.IndexAny(host, "/?#"); i >= 0 {
				host = host[:i]
			}
			if i := strings.LastIndex(host, "@"); i >= 0 { // strip user:pass@
				host = host[i+1:]
			}
			if reason, blocked := blockedIPOrHost(host); blocked {
				return "", fmt.Errorf("refusing request to %s: %s", host, reason)
			}

			var body io.Reader
			if b, _ := args["body"].(string); b != "" {
				body = strings.NewReader(b)
			}

			reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(reqCtx, method, targetURL, body)
			if err != nil {
				return "", fmt.Errorf("invalid request: %w", err)
			}
			req.Header.Set("User-Agent", "ShellSage-CLI/4.0")
			if hm, ok := args["headers"].(map[string]interface{}); ok {
				for k, v := range hm {
					if vs, ok := v.(string); ok {
						req.Header.Set(k, vs)
					}
				}
			}

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				return "", fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
			truncated := false
			if len(raw) >= 64*1024 {
				truncated = true
			}

			out := &bytes.Buffer{}
			fmt.Fprintf(out, "HTTP %s\n", resp.Status)
			for _, h := range []string{"Content-Type", "Location", "Retry-After", "X-Request-Id", "Server"} {
				if v := resp.Header.Get(h); v != "" {
					fmt.Fprintf(out, "%s: %s\n", h, v)
				}
			}
			out.WriteString("\n")

			pretty := raw
			var prettyBuf bytes.Buffer
			if strings.Contains(resp.Header.Get("Content-Type"), "json") {
				if err := json.Indent(&prettyBuf, raw, "", "  "); err == nil {
					pretty = prettyBuf.Bytes()
				}
			}
			out.Write(pretty)
			if truncated {
				out.WriteString("\n... (response body truncated at 64KB)")
			}
			return out.String(), nil
		},
	})
}

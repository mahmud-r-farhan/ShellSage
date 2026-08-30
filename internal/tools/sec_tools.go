package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"shellsage/internal/security"
)

func (r *ToolRegistry) registerSecurityTools() {
	// Security Headers Audit
	r.Register(ToolDef{
		Name:        "sec_headers",
		Description: "Audits HTTP security headers (HSTS, CSP, X-Frame-Options, Cookie security flags) for a web application URL",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{"type": "string", "description": "Target URL to analyze"},
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

			res, err := security.AuditSecurityHeaders(ctx, targetURL)
			if err != nil {
				return "", err
			}

			data, _ := json.MarshalIndent(res, "", "  ")
			return string(data), nil
		},
	})

	// SSL/TLS Certificate Audit
	r.Register(ToolDef{
		Name:        "sec_ssl",
		Description: "Inspects live SSL/TLS certificate validity, expiration date, cipher suite, and TLS protocol version",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"domain": map[string]interface{}{"type": "string", "description": "Target domain or hostname"},
			},
			"required": []string{"domain"},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			domain, _ := args["domain"].(string)
			if domain == "" {
				domain, _ = args["input"].(string)
			}
			if domain == "" {
				return "", fmt.Errorf("missing 'domain' parameter")
			}

			res, err := security.InspectSSLCertificate(domain)
			if err != nil {
				return "", err
			}

			data, _ := json.MarshalIndent(res, "", "  ")
			return string(data), nil
		},
	})

	// SAST Code Vulnerability Scan
	r.Register(ToolDef{
		Name:        "sec_sast",
		Description: "Performs Static Application Security Testing (SAST) scanning source code files for leaked API keys, credentials, SQL injection, and insecure crypto",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "Root directory or file to scan (default: current directory '.')"},
			},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			path, _ := args["path"].(string)
			if path == "" {
				path = "."
			}

			res, err := security.ScanCodebase(path)
			if err != nil {
				return "", err
			}

			data, _ := json.MarshalIndent(res, "", "  ")
			return string(data), nil
		},
	})
}

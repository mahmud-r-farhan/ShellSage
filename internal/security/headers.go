package security

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HeaderAuditResult represents the evaluation of HTTP security headers
type HeaderAuditResult struct {
	URL            string            `json:"url"`
	StatusCode     int               `json:"status_code"`
	Grade          string            `json:"grade"`
	Score          int               `json:"score"`
	PresentHeaders map[string]string `json:"present_headers"`
	MissingHeaders []string          `json:"missing_headers"`
	Warnings       []string          `json:"warnings"`
	Cookies        []string          `json:"cookies"`
}

// Recommended security headers
var requiredSecurityHeaders = []struct {
	Name        string
	Description string
	Points      int
}{
	{"Strict-Transport-Security", "Enforces HTTPS connections (HSTS)", 20},
	{"Content-Security-Policy", "Prevents XSS and malicious script injection (CSP)", 25},
	{"X-Frame-Options", "Prevents clickjacking attacks (DENY or SAMEORIGIN)", 15},
	{"X-Content-Type-Options", "Prevents MIME-type sniffing (nosniff)", 15},
	{"Referrer-Policy", "Controls referrer information in requests", 15},
	{"Permissions-Policy", "Restricts browser features and APIs", 10},
}

// AuditSecurityHeaders inspects a URL's HTTP response headers for defensive security posture
func AuditSecurityHeaders(ctx context.Context, targetURL string) (*HeaderAuditResult, error) {
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("stopped after 5 redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	req.Header.Set("User-Agent", "ShellSage-SecurityAuditor/3.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	result := &HeaderAuditResult{
		URL:            targetURL,
		StatusCode:     resp.StatusCode,
		PresentHeaders: make(map[string]string),
		MissingHeaders: make([]string, 0),
		Warnings:       make([]string, 0),
		Cookies:        make([]string, 0),
	}

	score := 0
	for _, h := range requiredSecurityHeaders {
		val := resp.Header.Get(h.Name)
		if val != "" {
			result.PresentHeaders[h.Name] = val
			score += h.Points
		} else {
			result.MissingHeaders = append(result.MissingHeaders, fmt.Sprintf("%s (%s)", h.Name, h.Description))
		}
	}

	// Check Server/X-Powered-By leakage
	if srv := resp.Header.Get("Server"); srv != "" {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Information Disclosure: 'Server: %s' header exposes backend software version", srv))
	}
	if pwr := resp.Header.Get("X-Powered-By"); pwr != "" {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Information Disclosure: 'X-Powered-By: %s' exposes framework details", pwr))
	}

	// Check Cookies flags
	for _, cookie := range resp.Cookies() {
		cookieIssues := []string{}
		if !cookie.Secure {
			cookieIssues = append(cookieIssues, "Missing 'Secure' flag")
		}
		if !cookie.HttpOnly {
			cookieIssues = append(cookieIssues, "Missing 'HttpOnly' flag")
		}
		if cookie.SameSite == http.SameSiteDefaultMode {
			cookieIssues = append(cookieIssues, "Missing 'SameSite' attribute")
		}
		cookieSummary := fmt.Sprintf("Cookie '%s'", cookie.Name)
		if len(cookieIssues) > 0 {
			cookieSummary += fmt.Sprintf(" -> Issues: %s", strings.Join(cookieIssues, ", "))
		} else {
			cookieSummary += " -> [Hardened]"
		}
		result.Cookies = append(result.Cookies, cookieSummary)
	}

	result.Score = score
	switch {
	case score >= 90:
		result.Grade = "A+"
	case score >= 75:
		result.Grade = "B"
	case score >= 50:
		result.Grade = "C"
	case score >= 30:
		result.Grade = "D"
	default:
		result.Grade = "F"
	}

	return result, nil
}

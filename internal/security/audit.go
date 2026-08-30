package security

import (
	"context"
	"fmt"
	"strings"
)

// RunFullAudit runs a comprehensive security posture assessment
func RunFullAudit(ctx context.Context, target string) string {
	var sb strings.Builder
	sb.WriteString("🛡️  ━━━━━━━━ ShellSage Security & Vulnerability Audit ━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("Target: %s\n\n", target))

	// Check if target is a web URL or domain
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.Contains(target, ".") {
		// 1. Headers Audit
		headerRes, err := AuditSecurityHeaders(ctx, target)
		if err == nil {
			sb.WriteString(fmt.Sprintf("📊 HTTP Security Headers: Grade %s (Score: %d/100)\n", headerRes.Grade, headerRes.Score))
			if len(headerRes.MissingHeaders) > 0 {
				sb.WriteString("   ⚠️  Missing Recommended Security Headers:\n")
				for _, m := range headerRes.MissingHeaders {
					sb.WriteString(fmt.Sprintf("      • %s\n", m))
				}
			}
			if len(headerRes.Warnings) > 0 {
				sb.WriteString("   ⚠️  Header Warnings:\n")
				for _, w := range headerRes.Warnings {
					sb.WriteString(fmt.Sprintf("      • %s\n", w))
				}
			}
			sb.WriteString("\n")
		}

		// 2. SSL/TLS Audit
		sslRes, err := InspectSSLCertificate(target)
		if err == nil {
			validStatus := "VALID"
			if !sslRes.Valid {
				validStatus = "INVALID / WARNING"
			}
			sb.WriteString(fmt.Sprintf("🔒 SSL/TLS Certificate: %s (%s)\n", validStatus, sslRes.TLSVersion))
			sb.WriteString(fmt.Sprintf("   • Issuer: %s | Expires in %d days (%s)\n", sslRes.Issuer, sslRes.DaysRemaining, sslRes.NotAfter.Format("2006-01-02")))
			if len(sslRes.Warnings) > 0 {
				for _, w := range sslRes.Warnings {
					sb.WriteString(fmt.Sprintf("   ⚠️  %s\n", w))
				}
			}
			sb.WriteString("\n")
		}
	}

	// 3. SAST Codebase Scan
	scanPath := target
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		scanPath = "."
	}
	sastRes, err := ScanCodebase(scanPath)
	if err == nil {
		sb.WriteString(fmt.Sprintf("🔍 SAST Code Vulnerability Scan (%d files analyzed):\n", sastRes.TotalFiles))
		if len(sastRes.Findings) == 0 {
			sb.WriteString("   ✅ No high/critical security vulnerabilities or leaked secrets found!\n")
		} else {
			sb.WriteString(fmt.Sprintf("   ⚠️  Identified %d findings (Critical: %d, High: %d, Medium: %d):\n",
				len(sastRes.Findings), sastRes.CriticalCount, sastRes.HighCount, sastRes.MediumCount))
			for i, f := range sastRes.Findings {
				if i >= 10 {
					sb.WriteString(fmt.Sprintf("   ... and %d more findings\n", len(sastRes.Findings)-10))
					break
				}
				sb.WriteString(fmt.Sprintf("   [%s] %s:%d - %s\n       Snippet: %s\n       Fix: %s\n",
					f.Severity, f.FilePath, f.LineNumber, f.Description, f.Snippet, f.Remediation))
			}
		}
	}

	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	return sb.String()
}

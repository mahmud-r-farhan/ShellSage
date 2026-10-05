package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"shellsage/internal/security"
)

func printJSON(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		color.Red("failed to encode JSON: %v", err)
		return
	}
	fmt.Println(string(data))
}

// RunFullAuditReport wraps the security package's combined audit.
func RunFullAuditReport(ctx context.Context, target string) string {
	return security.RunFullAudit(ctx, target)
}

// secFlagOpts carries global flags for `shellsage sec`.
type secFlagOpts struct {
	json bool
}

// SecReport implements the `sec` subcommand and backs the REPL's /sec command.
// args example: ["headers", "https://example.com"]
func SecReport(ctx context.Context, args []string) int {
	if len(args) == 0 {
		color.Yellow("Usage: shellsage sec <headers|ssl|ports|sast|audit> [target]\n")
		color.White("  headers <url>   Audit HTTP security headers & cookie flags (graded)\n")
		color.White("  ssl <domain>    Inspect SSL/TLS certificate, protocol & cipher\n")
		color.White("  ports <host>    Probe common dev/infra service ports\n")
		color.White("  sast [path]     Static scan for leaked secrets & vulnerable patterns\n")
		color.White("  audit <target>  Full combined posture audit (also: `shellsage audit`)\n")
		return 2
	}

	fs := flag.NewFlagSet("sec", flag.ContinueOnError)
	var opts secFlagOpts
	fs.BoolVar(&opts.json, "json", false, "machine-readable JSON output")
	rest := args[1:]
	// Pre-scan for --json so positional args stay simple.
	var filtered []string
	for _, a := range rest {
		if a == "--json" {
			opts.json = true
			continue
		}
		filtered = append(filtered, a)
	}
	rest = filtered

	sub := strings.ToLower(args[0])
	target := "."
	if len(rest) > 0 {
		target = rest[0]
	}

	switch sub {
	case "headers":
		if len(rest) == 0 {
			color.Yellow("Usage: shellsage sec headers <url>\n")
			return 2
		}
		res, err := security.AuditSecurityHeaders(ctx, target)
		if err != nil {
			color.Red("❌ Audit failed: %v\n", err)
			return 1
		}
		if opts.json {
			printJSON(res)
			return 0
		}
		color.Green("📊 Grade: %s  (Security Score: %d/100)\n\n", res.Grade, res.Score)
		if len(res.PresentHeaders) > 0 {
			color.White("  ✅ Present Security Headers:\n")
			for k, v := range res.PresentHeaders {
				color.HiGreen("     • %s: %s\n", k, trunc(v, 60))
			}
		}
		if len(res.MissingHeaders) > 0 {
			color.White("\n  ⚠️  Missing Headers:\n")
			for _, m := range res.MissingHeaders {
				color.HiYellow("     • %s\n", m)
			}
		}
		if len(res.Warnings) > 0 {
			color.White("\n  ⚠️  Information Disclosure Warnings:\n")
			for _, w := range res.Warnings {
				color.HiRed("     • %s\n", w)
			}
		}
		if len(res.Cookies) > 0 {
			color.White("\n  🍪 Cookies Security Analysis:\n")
			for _, c := range res.Cookies {
				color.Cyan("     • %s\n", c)
			}
		}
		fmt.Println()
		return 0

	case "ssl":
		if len(rest) == 0 {
			color.Yellow("Usage: shellsage sec ssl <domain>\n")
			return 2
		}
		res, err := security.InspectSSLCertificate(target)
		if err != nil {
			color.Red("❌ Inspection failed: %v\n", err)
			return 1
		}
		if opts.json {
			printJSON(res)
			return 0
		}
		validTag := color.GreenString("VALID")
		if !res.Valid {
			validTag = color.RedString("INVALID / WARNING")
		}
		color.White("  Status:    %s\n", validTag)
		color.White("  Subject:   %s\n", res.Subject)
		color.White("  Issuer:    %s\n", res.Issuer)
		color.White("  Protocol:  %s\n", res.TLSVersion)
		color.White("  Cipher:    %s\n", res.CipherSuite)
		color.White("  Validity:  %s → %s (%d days remaining)\n",
			res.NotBefore.Format("2006-01-02"), res.NotAfter.Format("2006-01-02"), res.DaysRemaining)
		for _, w := range res.Warnings {
			color.HiRed("  ⚠️  %s\n", w)
		}
		fmt.Println()
		return 0

	case "ports":
		if len(rest) == 0 {
			target = "localhost"
		}
		ports := security.AuditPortConnectivity(target, nil)
		openCount := 0
		color.Cyan("🔍 Service port audit for %s:\n\n", target)
		for _, p := range ports {
			if p.IsOpen {
				openCount++
				color.HiGreen("  🟢 Port %-5d [%-24s] : OPEN  %s\n", p.Port, p.Service, p.Banner)
			}
		}
		if openCount == 0 {
			color.White("  🔒 No standard exposed ports detected.\n")
		}
		fmt.Println()
		return 0

	case "sast":
		res, err := security.ScanCodebase(target)
		if err != nil {
			color.Red("❌ SAST scan failed: %v\n", err)
			return 1
		}
		if opts.json {
			printJSON(res)
			if res.CriticalCount > 0 {
				return 1
			}
			return 0
		}
		color.White("  Total Files Scanned: %d\n", res.TotalFiles)
		if len(res.Findings) == 0 {
			color.Green("  ✅ Clean! No leaked secrets or high-severity vulnerabilities found.\n")
			return 0
		}
		color.Yellow("  ⚠️  %d findings (Critical: %d, High: %d, Medium: %d):\n\n",
			len(res.Findings), res.CriticalCount, res.HighCount, res.MediumCount)
		for i, f := range res.Findings {
			sevColor := color.HiYellowString
			if f.Severity == "CRITICAL" || f.Severity == "HIGH" {
				sevColor = color.HiRedString
			}
			color.White("  [%s] %s:%d\n    Issue: %s\n    Code:  %s\n    Fix:   %s\n\n",
				sevColor(f.Severity), f.FilePath, f.LineNumber, f.Description, color.HiBlackString(trunc(f.Snippet, 90)), color.CyanString(f.Remediation))
			if i >= 19 {
				color.HiBlack("  ... and %d more findings\n", len(res.Findings)-20)
				break
			}
		}
		if res.CriticalCount > 0 {
			return 1
		}
		return 0

	case "audit":
		fmt.Print(security.RunFullAudit(ctx, target))
		return 0
	}

	color.Red("Unknown security command %q (headers|ssl|ports|sast|audit)\n", sub)
	return 2
}

func trunc(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

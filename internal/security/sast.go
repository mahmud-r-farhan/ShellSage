package security

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// VulnerabilityFinding represents an identified security weakness or leak
type VulnerabilityFinding struct {
	RuleID      string `json:"rule_id"`
	Severity    string `json:"severity"` // CRITICAL, HIGH, MEDIUM, LOW
	Description string `json:"description"`
	FilePath    string `json:"file_path"`
	LineNumber  int    `json:"line_number"`
	Snippet     string `json:"snippet"`
	Remediation string `json:"remediation"`
}

// SASTAuditResult represents the complete codebase scan report
type SASTAuditResult struct {
	ScannedPath  string                 `json:"scanned_path"`
	TotalFiles   int                    `json:"total_files"`
	Findings     []VulnerabilityFinding `json:"findings"`
	CriticalCount int                    `json:"critical_count"`
	HighCount     int                    `json:"high_count"`
	MediumCount   int                    `json:"medium_count"`
	LowCount      int                    `json:"low_count"`
}

type sastRule struct {
	ID          string
	Severity    string
	Description string
	Pattern     *regexp.Regexp
	Remediation string
}

var sastRules = []sastRule{
	{
		ID:          "SEC-SECRET-OPENAI",
		Severity:    "CRITICAL",
		Description: "Hardcoded OpenAI API Key detected",
		Pattern:     regexp.MustCompile(`(?i)sk-(?:proj-)?[A-Za-z0-9_-]{20,}`),
		Remediation: "Store API keys in environment variables or a secure secret manager; never commit keys to Git.",
	},
	{
		ID:          "SEC-SECRET-AWS",
		Severity:    "CRITICAL",
		Description: "Hardcoded AWS Access Key ID detected",
		Pattern:     regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		Remediation: "Use IAM roles or AWS Vault; remove hardcoded AWS credentials immediately.",
	},
	{
		ID:          "SEC-SECRET-GENERIC-KEY",
		Severity:    "HIGH",
		Description: "Potential hardcoded API Secret or Token",
		Pattern:     regexp.MustCompile(`(?i)(?:api_key|apikey|secret_key|private_key|token|auth_token)\s*[:=]\s*["'][A-Za-z0-9_\-\.]{12,}["']`),
		Remediation: "Extract secret values to .env configuration files and ensure .env is git-ignored.",
	},
	{
		ID:          "SEC-SECRET-PRIVATE-KEY",
		Severity:    "CRITICAL",
		Description: "Embedded RSA / OpenSSH Private Key Block",
		Pattern:     regexp.MustCompile(`-----BEGIN (?:RSA|OPENSSH|EC|DSA)? PRIVATE KEY-----`),
		Remediation: "Revoke this private key and remove it from repository history immediately.",
	},
	{
		ID:          "SEC-INJ-SQL",
		Severity:    "HIGH",
		Description: "Potential SQL Injection via string formatting/concatenation",
		Pattern:     regexp.MustCompile(`(?i)(?:db\.Query|db\.Exec|SELECT\s+.*FROM|INSERT\s+INTO|UPDATE\s+.*SET)\s*\([^)]*(?:\+|fmt\.Sprintf)`),
		Remediation: "Use parameterized queries ($1, ? or named parameters) instead of string formatting.",
	},
	{
		ID:          "SEC-INJ-CMD",
		Severity:    "HIGH",
		Description: "Unsafe Command Execution via shell interpreter",
		Pattern:     regexp.MustCompile(`(?i)(?:exec\.Command\s*\(\s*["'](?:cmd|sh|bash)["']\s*,\s*["'](?:/C|-c)["']|system\s*\()`),
		Remediation: "Pass arguments as separate array elements to exec.Command rather than spawning full shells.",
	},
	{
		ID:          "SEC-CRYPTO-MD5-SHA1",
		Severity:    "MEDIUM",
		Description: "Use of weak cryptographic hash (MD5 / SHA1)",
		Pattern:     regexp.MustCompile(`(?i)(?:md5\.New|md5\.Sum|sha1\.New|sha1\.Sum)`),
		Remediation: "Upgrade to SHA-256 (crypto/sha256), SHA-512, or argon2id/bcrypt for password hashing.",
	},
	{
		ID:          "SEC-AUTH-JWT-NONE",
		Severity:    "HIGH",
		Description: "Potential insecure JWT 'none' algorithm or missing signature check",
		Pattern:     regexp.MustCompile(`(?i)(?:jwt\.SigningMethodNone|"alg"\s*:\s*"none")`),
		Remediation: "Enforce explicit cryptographic signing algorithms (e.g. Ed25519, RS256, HS256).",
	},
}

// ScanCodebase performs SAST scanning on a target directory or file
func ScanCodebase(rootPath string) (*SASTAuditResult, error) {
	if rootPath == "" {
		rootPath = "."
	}

	result := &SASTAuditResult{
		ScannedPath: rootPath,
		Findings:    make([]VulnerabilityFinding, 0),
	}

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || name == "dist" || name == ".gemini" {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip binaries or images
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".exe" || ext == ".dll" || ext == ".png" || ext == ".jpg" || ext == ".zip" || ext == ".tar" || ext == ".pdf" {
			return nil
		}

		result.TotalFiles++
		scanFile(path, result)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error during SAST scan: %w", err)
	}

	for _, f := range result.Findings {
		switch f.Severity {
		case "CRITICAL":
			result.CriticalCount++
		case "HIGH":
			result.HighCount++
		case "MEDIUM":
			result.MediumCount++
		case "LOW":
			result.LowCount++
		}
	}

	return result, nil
}

func scanFile(path string, result *SASTAuditResult) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 1

	for scanner.Scan() {
		line := scanner.Text()

		// Skip test rule definitions themselves
		if strings.Contains(path, "sast.go") || strings.Contains(path, "sast_test.go") {
			lineNum++
			continue
		}

		for _, rule := range sastRules {
			if rule.Pattern.MatchString(line) {
				snippet := strings.TrimSpace(line)
				if len(snippet) > 100 {
					snippet = snippet[:100] + "..."
				}

				result.Findings = append(result.Findings, VulnerabilityFinding{
					RuleID:      rule.ID,
					Severity:    rule.Severity,
					Description: rule.Description,
					FilePath:    path,
					LineNumber:  lineNum,
					Snippet:     snippet,
					Remediation: rule.Remediation,
				})
			}
		}
		lineNum++
	}
}

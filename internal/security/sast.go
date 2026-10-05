package security

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"math"
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
	ScannedPath   string                 `json:"scanned_path"`
	TotalFiles    int                    `json:"total_files"`
	Findings      []VulnerabilityFinding `json:"findings"`
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
	// Guard optionally vetoes a match (used to reduce false positives).
	Guard func(line string) bool
}

var reGenericSecretValue = regexp.MustCompile(`["']([A-Za-z0-9_\-\./+=]{12,})["']`)

func genericSecretGuard(line string) bool {
	for _, m := range reGenericSecretValue.FindAllStringSubmatch(line, -1) {
		if looksLikeRealSecret(m[1]) {
			return true
		}
	}
	return false
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
		Guard:       genericSecretGuard,
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
	{
		ID:          "SEC-SECRET-GITHUB",
		Severity:    "CRITICAL",
		Description: "Hardcoded GitHub token detected (classic or fine-grained PAT / OAuth)",
		Pattern:     regexp.MustCompile(`(?:ghp_[0-9A-Za-z]{36}|github_pat_[0-9A-Za-z_]{60,}|gho_[0-9A-Za-z]{36}|ghs_[0-9A-Za-z]{36}|ghu_[0-9A-Za-z]{36})`),
		Remediation: "Revoke the token at github.com/settings/tokens; use `gh auth` or CI OIDC federation instead.",
	},
	{
		ID:          "SEC-SECRET-SLACK",
		Severity:    "HIGH",
		Description: "Hardcoded Slack API/bot token detected",
		Pattern:     regexp.MustCompile(`xox[baprs]-[0-9A-Za-z-]{10,}`),
		Remediation: "Revoke in the Slack app console and move the token to a secret manager.",
	},
	{
		ID:          "SEC-SECRET-STRIPE",
		Severity:    "CRITICAL",
		Description: "Hardcoded Stripe secret key detected",
		Pattern:     regexp.MustCompile(`(?:sk|rk)_live_[0-9A-Za-z]{16,}`),
		Remediation: "Roll the key in the Stripe dashboard immediately; live keys must never be committed.",
	},
	{
		ID:          "SEC-SECRET-GOOGLE",
		Severity:    "HIGH",
		Description: "Hardcoded Google API key detected",
		Pattern:     regexp.MustCompile(`AIza[0-9A-Za-z_\-]{35}`),
		Remediation: "Rotate via Google Cloud console and restrict the key by application/API.",
	},
	{
		ID:          "SEC-SSRF-FETCH",
		Severity:    "MEDIUM",
		Description: "User-controlled URL used directly in an HTTP request (potential SSRF)",
		Pattern:     regexp.MustCompile(`(?i)(?:http\.Get|http\.NewRequest\w*)\s*\([^)]*(?:\+\s*\w+|fmt\.Sprintf|url\s*\+\s*|\buserInput)`),
		Remediation: "Validate scheme/host, block link-local & metadata IPs (169.254.169.254), and require an allowlist where feasible.",
	},
	{
		ID:          "SEC-DESERIALIZE-UNSAFE",
		Severity:    "MEDIUM",
		Description: "unsafe/deserialization-related anti-patterns: pickle or gob.Decode on remote data",
		Pattern:     regexp.MustCompile(`(?i)(?:pickle\.loads?\s*\(|gob\.NewDecoder\s*\(\s*(?:resp|conn|r)\.Body|yaml\.Unmarshal\s*\([^)]*\b(?:body|payload|raw|data)\b)`),
		Remediation: "Use JSON with explicit schemas or yaml.SafeLoader; never deserialize untrusted bytes.",
	},
	{
		ID:          "SEC-TLS-INSECURE-SKIP",
		Severity:    "HIGH",
		Description: "TLS certificate verification disabled (InsecureSkipVerify: true)",
		Pattern:     regexp.MustCompile(`InsecureSkipVerify\s*:\s*true`),
		Remediation: "Pin the CA bundle or install proper certs; skipping verification enables MITR for all traffic.",
		Guard:       func(line string) bool { return !strings.Contains(line, "#nosec") },
	},
}

var placeholderPattern = regexp.MustCompile(`(?i)(?:changeme|your[_-]?(?:api)?[_-]?(?:key|token|secret)|placeholder|example|dummy|[^a-z0-9]test|xxxx|<[^>]+>|\$\{|\%\(|here\b)`)

// looksLikeRealSecret filters obvious placeholder values out of the generic
// secret rule to keep SAST output trustworthy (few false positives => CI-gateable).
func looksLikeRealSecret(v string) bool {
	if len(v) < 16 {
		return false
	}
	if placeholderPattern.MatchString(v) {
		return false
	}
	// Shannon entropy over the alphabet used; real keys are high-entropy.
	freq := map[byte]int{}
	for i := 0; i < len(v); i++ {
		freq[v[i]]++
	}
	h := 0.0
	for _, c := range freq {
		p := float64(c) / float64(len(v))
		h -= p * math.Log2(p)
	}
	return h > 3.0
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

	// Binary sniff: compiled artifacts (e.g. this very CLI) contain source-like
	// strings from their standard library and must not be pattern-matched.
	probe := make([]byte, 512)
	if n, _ := file.Read(probe); bytes.IndexByte(probe[:n], 0) >= 0 {
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return
	}

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
			if rule.Pattern.MatchString(line) && (rule.Guard == nil || rule.Guard(line)) {
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

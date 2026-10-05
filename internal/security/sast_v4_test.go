package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSASTNewSecretRules(t *testing.T) {
	dir := t.TempDir()
	content := `package main
var gh = "ghp_123456789012345678901234567890abcdefgh"
var slack = "xoxb-1234567890-abcdef-XYZxyz"
var skip = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
`
	if err := os.WriteFile(filepath.Join(dir, "secrets.go"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	res, err := ScanCodebase(dir)
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, f := range res.Findings {
		joined += f.RuleID + "\n"
	}
	for _, want := range []string{"SEC-SECRET-GITHUB", "SEC-SECRET-SLACK", "SEC-TLS-INSECURE-SKIP"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected rule %s to fire, findings:\n%s", want, joined)
		}
	}
}

func TestSASTPlaceholderNotFlagged(t *testing.T) {
	dir := t.TempDir()
	content := `package main
var apiKey = "your_api_key_here"
var token = "changeme"
`
	if err := os.WriteFile(filepath.Join(dir, "config.go"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	res, err := ScanCodebase(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range res.Findings {
		if f.RuleID == "SEC-SECRET-GENERIC-KEY" {
			t.Errorf("placeholder value flagged as secret: %+v", f)
		}
	}
}

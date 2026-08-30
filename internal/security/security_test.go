package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecurityHeadersAudit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	res, err := AuditSecurityHeaders(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("failed to audit headers: %v", err)
	}

	if len(res.PresentHeaders) != 3 {
		t.Errorf("expected 3 present headers, got %d", len(res.PresentHeaders))
	}
	if res.Score < 50 {
		t.Errorf("expected score >= 50, got %d", res.Score)
	}
}

func TestSASTScan(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "vulnerable.go")
	badCode := `package main
var apiKey = "sk-proj-12345678901234567890"
func runQuery(dbQuery string) {
    db.Query("SELECT * FROM users WHERE name = " + fmt.Sprintf("%s", userInput))
}
`
	err := os.WriteFile(testFile, []byte(badCode), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	res, err := ScanCodebase(tempDir)
	if err != nil {
		t.Fatalf("failed to run SAST scan: %v", err)
	}

	if len(res.Findings) < 2 {
		t.Errorf("expected at least 2 findings (API key and SQL injection), got %d", len(res.Findings))
	}
}

func TestPortAudit(t *testing.T) {
	// Scan localhost on a non-listening high port
	ports := AuditPortConnectivity("127.0.0.1", []int{59999})
	if len(ports) != 1 {
		t.Errorf("expected 1 port result, got %d", len(ports))
	}
}

func TestRunFullAudit(t *testing.T) {
	tempDir := t.TempDir()
	out := RunFullAudit(context.Background(), tempDir)
	if !strings.Contains(out, "Security & Vulnerability Audit") {
		t.Errorf("audit output missing header: %s", out)
	}
}

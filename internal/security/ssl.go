package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"strings"
	"time"
)

// SSLAuditResult represents the SSL/TLS certificate verification result
type SSLAuditResult struct {
	Domain             string    `json:"domain"`
	Valid              bool      `json:"valid"`
	Subject            string    `json:"subject"`
	Issuer             string    `json:"issuer"`
	NotBefore          time.Time `json:"not_before"`
	NotAfter           time.Time `json:"not_after"`
	DaysRemaining      int       `json:"days_remaining"`
	DNSNames           []string  `json:"dns_names"`
	TLSVersion         string    `json:"tls_version"`
	CipherSuite        string    `json:"cipher_suite"`
	SignatureAlgorithm string    `json:"signature_algorithm"`
	Warnings           []string  `json:"warnings"`
}

// InspectSSLCertificate performs live SSL/TLS certificate analysis on a domain
func InspectSSLCertificate(domain string) (*SSLAuditResult, error) {
	// Clean domain string
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}
	if !strings.Contains(domain, ":") {
		domain = domain + ":443"
	}

	host, _, err := net.SplitHostPort(domain)
	if err != nil {
		host = domain
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}

	// This is a *collector*: the handshake fetches the presented chain without
	// letting Go fail fast, so we can always report details. The trust decision
	// itself is made explicitly below with x509.Verify against the system root
	// pool — never silently skipped.
	conn, err := tls.DialWithDialer(dialer, "tcp", domain, &tls.Config{
		ServerName: host,
		// Verification is performed manually via verifyPeerChain below.
		InsecureSkipVerify: true, // #nosec G402 -- collector; trust checked explicitly after handshake
	})
	if err != nil {
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}
	defer conn.Close()

	state := conn.ConnectionState()
	warnings := verifyPeerChain(host, state)
	return parseCertState(host, state, warnings), nil
}

// verifyPeerChain performs the real trust evaluation offline: chain to a system
// root and hostname match, returning human-readable failures as warnings.
func verifyPeerChain(host string, state tls.ConnectionState) []string {
	if len(state.PeerCertificates) == 0 {
		return []string{"No peer certificates provided by server"}
	}
	roots, err := x509.SystemCertPool()
	if err != nil || roots == nil {
		return []string{"Unable to load system root CA pool — certificate trust could not be verified"}
	}
	intermediates := x509.NewCertPool()
	for _, c := range state.PeerCertificates[1:] {
		intermediates.AddCert(c)
	}
	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		DNSName:       host,
	}
	if _, err := state.PeerCertificates[0].Verify(opts); err != nil {
		return []string{fmt.Sprintf("Certificate Validation Failed: %v", err)}
	}
	return nil
}

func parseCertState(host string, state tls.ConnectionState, initialWarnings []string) *SSLAuditResult {
	res := &SSLAuditResult{
		Domain:   host,
		Valid:    len(initialWarnings) == 0,
		Warnings: initialWarnings,
	}

	if len(state.PeerCertificates) == 0 {
		res.Warnings = append(res.Warnings, "No peer certificates provided by server")
		return res
	}

	cert := state.PeerCertificates[0]
	res.Subject = cert.Subject.CommonName
	res.Issuer = cert.Issuer.CommonName
	if res.Issuer == "" {
		res.Issuer = cert.Issuer.Organization[0]
	}
	res.NotBefore = cert.NotBefore
	res.NotAfter = cert.NotAfter
	res.DaysRemaining = int(time.Until(cert.NotAfter).Hours() / 24)
	res.DNSNames = cert.DNSNames
	res.SignatureAlgorithm = cert.SignatureAlgorithm.String()

	// Check expiration warnings
	if res.DaysRemaining <= 0 {
		res.Valid = false
		res.Warnings = append(res.Warnings, "CRITICAL: Certificate has expired!")
	} else if res.DaysRemaining < 30 {
		res.Warnings = append(res.Warnings, fmt.Sprintf("WARNING: Certificate expires in %d days", res.DaysRemaining))
	}

	// Map TLS Version
	switch state.Version {
	case tls.VersionTLS13:
		res.TLSVersion = "TLS 1.3 (Modern & Secure)"
	case tls.VersionTLS12:
		res.TLSVersion = "TLS 1.2 (Standard)"
	case tls.VersionTLS11:
		res.TLSVersion = "TLS 1.1 (Deprecated)"
		res.Warnings = append(res.Warnings, "Server supports outdated TLS 1.1 protocol")
	case tls.VersionTLS10:
		res.TLSVersion = "TLS 1.0 (Insecure)"
		res.Warnings = append(res.Warnings, "Server supports insecure TLS 1.0 protocol")
	default:
		res.TLSVersion = fmt.Sprintf("TLS Version %x", state.Version)
	}

	res.CipherSuite = tls.CipherSuiteName(state.CipherSuite)

	return res
}

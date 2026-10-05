package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HTTP transport shared utilities: retry with exponential backoff + jitter for
// transient failures (rate limits, gateway hiccups, dead VPNs), which makes
// ShellSage survivable in real-world CI and flaky-network scenarios.

const (
	defaultMaxRetries = 3
	backoffBase       = 250 * time.Millisecond
	backoffCap        = 8 * time.Second
)

// retryableStatus reports whether an HTTP status code is worth retrying.
func retryableStatus(code int) bool {
	switch code {
	case http.StatusRequestTimeout, // 408
		http.StatusTooEarly,        // 425
		http.StatusTooManyRequests: // 429
		return true
	}
	return code >= 500 // 5xx incl. CF edge codes 52x
}

// retryableNetError reports whether a transport error looks transient.
func retryableNetError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true // connection refused/reset etc.: cheap to retry with backoff
	}
	return false
}

// retryAfterDelay parses Retry-After (delta-seconds form) if present.
func retryAfterDelay(resp *http.Response) (time.Duration, bool) {
	v := resp.Header.Get("Retry-After")
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && secs >= 0 {
		d := time.Duration(secs) * time.Second
		if d > 30*time.Second {
			d = 30 * time.Second // cap so a hostile header can't hang the CLI
		}
		return d, true
	}
	return 0, false
}

func backoffDelay(attempt int) time.Duration {
	d := backoffBase << uint(attempt)
	if d > backoffCap || d <= 0 {
		d = backoffCap
	}
	// Full jitter up to 1/4 of the delay to avoid thundering herds.
	d -= time.Duration(rand.Int63n(int64(d/4 + 1)))
	return d
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// doWithRetry executes an HTTP request, recreating it via makeReq on each attempt
// (request bodies are consumed on the first try, so makeReq must produce a fresh
// *http.Request every call). The final response is returned even on error status
// codes; callers decide how to format failures. maxRetries <= 0 disables retrying.
func doWithRetry(ctx context.Context, client *http.Client, maxRetries int, makeReq func() (*http.Request, error)) (*http.Response, error) {
	if maxRetries < 0 {
		maxRetries = 0
	}
	var lastErr error
	var lastResp *http.Response

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			if !sleepWithContext(ctx, backoffDelay(attempt-1)) {
				return nil, ctx.Err()
			}
		}

		req, err := makeReq()
		if err != nil {
			return nil, err
		}
		req = req.WithContext(ctx)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if !retryableNetError(err) || attempt == maxRetries {
				return nil, err
			}
			continue
		}

		if !retryableStatus(resp.StatusCode) || attempt == maxRetries {
			return resp, nil
		}

		// Drain & close so the connection can be reused.
		wait, hasWait := retryAfterDelay(resp)
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
		lastResp = nil
		lastErr = fmt.Errorf("http %d", resp.StatusCode)
		if hasWait && wait > 0 {
			if !sleepWithContext(ctx, wait) {
				return nil, ctx.Err()
			}
		}
	}
	// Unreachable in practice, but keeps the compiler happy.
	_ = lastResp
	return nil, fmt.Errorf("exhausted retries: %w", lastErr)
}

// formatAPIError turns an error HTTP response into a human-friendly message
// with actionable hints (this is the "tell me what I did wrong" path that
// distinguishes hobby CLIs from daily-drivers).
func formatAPIError(providerName string, resp *http.Response) error {
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	_ = resp.Body.Close()
	body := strings.TrimSpace(string(bodyBytes))
	if len(body) > 800 {
		body = body[:800] + "..."
	}

	hint := ""
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		hint = "\nHint: your API key looks invalid for this provider. Run `shellsage config wizard` or set the provider's *_API_KEY env var."
	case resp.StatusCode == http.StatusNotFound:
		hint = "\nHint: wrong base URL or model/deployment id. Check `shellsage provider show " + strings.ToLower(providerName) + "` and `shellsage models list --remote`."
	case resp.StatusCode == http.StatusTooManyRequests:
		hint = "\nHint: rate-limited (429) — ShellSage already retried with backoff; consider switching model/provider or adding cooldown."
	case resp.StatusCode >= 500:
		hint = "\nHint: provider-side outage — ShellSage retried; try again later or another provider (`shellsage --provider groq ask ...`)."
	}

	return fmt.Errorf("%s API error (status %d): %s%s", providerName, resp.StatusCode, body, hint)
}

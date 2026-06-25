package connectors

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// HTTPHealth is a generic, auth-less connector: it probes a URL and reports reachability,
// latency, and a human-readable message. It provides service.status only. Real application
// versions are left to typed connectors; here we surface the Server header as a hint.
type HTTPHealth struct {
	client *http.Client
}

// NewHTTPHealth builds the connector with a bounded HTTP client.
func NewHTTPHealth() *HTTPHealth {
	return &HTTPHealth{
		client: &http.Client{Timeout: 8 * time.Second},
	}
}

func (h *HTTPHealth) Type() string { return "http-health" }

func (h *HTTPHealth) Provides() []ResourceKind { return []ResourceKind{KindServiceStatus} }

// probe performs the GET and returns the derived status.
func (h *HTTPHealth) probe(ctx context.Context, cfg Config) ServiceStatus {
	if cfg.BaseURL == "" {
		return ServiceStatus{Up: false, Message: "no URL configured"}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.BaseURL, nil)
	if err != nil {
		return ServiceStatus{Up: false, Message: "invalid URL"}
	}

	start := time.Now()
	resp, err := h.client.Do(req)
	latency := int(time.Since(start).Milliseconds())
	if err != nil {
		return ServiceStatus{Up: false, LatencyMs: latency, Message: mapErr(err)}
	}
	defer resp.Body.Close()

	// Reachable + status < 500 counts as up; 4xx means "up but needs auth", 5xx means down.
	up := resp.StatusCode < 500
	msg := fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	if server := resp.Header.Get("Server"); server != "" {
		msg += " · " + server
	}
	return ServiceStatus{Up: up, LatencyMs: latency, Message: msg}
}

func (h *HTTPHealth) Test(ctx context.Context, cfg Config) (TestResult, error) {
	s := h.probe(ctx, cfg)
	return TestResult{
		OK:        s.Up,
		Version:   s.Version,
		Message:   s.Message,
		LatencyMs: s.LatencyMs,
	}, nil
}

func (h *HTTPHealth) Fetch(ctx context.Context, cfg Config, kind ResourceKind) (Resource, error) {
	if kind != KindServiceStatus {
		return Resource{}, fmt.Errorf("http-health does not provide %q", kind)
	}
	return Resource{Kind: KindServiceStatus, Data: h.probe(ctx, cfg)}, nil
}

// mapErr turns low-level network errors into friendly, actionable messages.
func mapErr(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "connection timed out"
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return "connection refused — is the service running?"
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "host not found — check the URL"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "connection timed out"
	}
	return "unreachable"
}

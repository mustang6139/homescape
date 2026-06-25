package connectors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func statusOf(t *testing.T, h *HTTPHealth, url string) ServiceStatus {
	t.Helper()
	res, err := h.Fetch(context.Background(), Config{BaseURL: url}, KindServiceStatus)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	s, ok := res.Data.(ServiceStatus)
	if !ok {
		t.Fatalf("expected ServiceStatus, got %T", res.Data)
	}
	return s
}

func TestHTTPHealthUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Server", "test-server")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := statusOf(t, NewHTTPHealth(), srv.URL)
	if !s.Up {
		t.Errorf("expected up, got %+v", s)
	}
	if s.LatencyMs < 0 {
		t.Errorf("latency should be >= 0, got %d", s.LatencyMs)
	}
}

func TestHTTPHealthAuthRequiredIsUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	// 401 means reachable but needs auth → still "up".
	if s := statusOf(t, NewHTTPHealth(), srv.URL); !s.Up {
		t.Errorf("401 should be up (reachable), got %+v", s)
	}
}

func TestHTTPHealthServerErrorIsDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	if s := statusOf(t, NewHTTPHealth(), srv.URL); s.Up {
		t.Errorf("502 should be down, got %+v", s)
	}
}

func TestHTTPHealthRefused(t *testing.T) {
	// Start then immediately close to get a refused connection on that port.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	s := statusOf(t, NewHTTPHealth(), url)
	if s.Up {
		t.Errorf("closed server should be down, got %+v", s)
	}
	if s.Message == "" {
		t.Error("expected a human-readable message on failure")
	}
}

func TestHTTPHealthNoURL(t *testing.T) {
	if s := statusOf(t, NewHTTPHealth(), ""); s.Up {
		t.Errorf("empty URL should be down, got %+v", s)
	}
}

func TestTestConnection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr, err := NewHTTPHealth().Test(context.Background(), Config{BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("test: %v", err)
	}
	if !tr.OK {
		t.Errorf("expected OK test result, got %+v", tr)
	}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry(NewHTTPHealth())
	if _, ok := r.Get("http-health"); !ok {
		t.Error("http-health should be registered")
	}
	if _, ok := r.Get("nope"); ok {
		t.Error("unknown type should not resolve")
	}
	if got := r.Types(); len(got) != 1 || got[0] != "http-health" {
		t.Errorf("Types() = %v, want [http-health]", got)
	}
}

func TestFetchUnsupportedKind(t *testing.T) {
	if _, err := NewHTTPHealth().Fetch(context.Background(), Config{BaseURL: "http://x"}, KindMediaSessions); err == nil {
		t.Error("expected error for unsupported kind")
	}
}

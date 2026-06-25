package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/MusiThang/homescape/internal/connectors"
	"github.com/MusiThang/homescape/internal/secret"
	"github.com/MusiThang/homescape/internal/server"
	"github.com/MusiThang/homescape/internal/store"
)

type emptySource struct{}

func (emptySource) ActiveTargets() ([]connectors.Target, error) { return nil, nil }

func newTestServer(t *testing.T, passphrase string) (*httptest.Server, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	// Seed a default scape so /api/scape works if needed.
	reg := connectors.NewRegistry(connectors.NewHTTPHealth())
	poller := connectors.NewPoller(reg, emptySource{}, time.Hour, nil, nil)
	vault := secret.New(passphrase, st.Secrets(), st.Settings())

	srv := server.New(server.Deps{
		Store:    st,
		Vault:    vault,
		Registry: reg,
		Poller:   poller,
		Hub:      server.NewHub(),
		WebFS:    fstest.MapFS{"index.html": {Data: []byte("<html></html>")}},
	})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, st
}

func do(t *testing.T, method, url string, body any) (*http.Response, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, url, rdr)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, data
}

func TestIntegrationCreateListDelete(t *testing.T) {
	ts, _ := newTestServer(t, "")

	// Create
	resp, body := do(t, http.MethodPost, ts.URL+"/api/integrations", map[string]string{
		"type": "http-health", "name": "My Service", "baseUrl": "http://x:80",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, body %s", resp.StatusCode, body)
	}
	var created map[string]any
	_ = json.Unmarshal(body, &created)
	if created["id"] != "my-service" {
		t.Errorf("id = %v, want my-service", created["id"])
	}
	if created["hasSecret"] != false {
		t.Errorf("hasSecret = %v, want false", created["hasSecret"])
	}

	// List
	resp, body = do(t, http.MethodGet, ts.URL+"/api/integrations", nil)
	var list []map[string]any
	_ = json.Unmarshal(body, &list)
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1 (%s)", len(list), body)
	}

	// Delete
	resp, _ = do(t, http.MethodDelete, ts.URL+"/api/integrations/my-service", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete status = %d", resp.StatusCode)
	}
	_, body = do(t, http.MethodGet, ts.URL+"/api/integrations", nil)
	_ = json.Unmarshal(body, &list)
	if len(list) != 0 {
		t.Errorf("after delete list len = %d, want 0", len(list))
	}
}

func TestIntegrationUnknownType(t *testing.T) {
	ts, _ := newTestServer(t, "")
	resp, _ := do(t, http.MethodPost, ts.URL+"/api/integrations", map[string]string{
		"type": "bogus", "name": "X",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unknown type status = %d, want 400", resp.StatusCode)
	}
}

func TestSecretRequiresKey(t *testing.T) {
	ts, _ := newTestServer(t, "") // vault disabled
	resp, _ := do(t, http.MethodPost, ts.URL+"/api/integrations", map[string]string{
		"type": "http-health", "name": "WithSecret", "secret": "abc",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("secret without key status = %d, want 400", resp.StatusCode)
	}
}

func TestSecretStoredWhenKeySet(t *testing.T) {
	ts, _ := newTestServer(t, "passphrase")
	resp, body := do(t, http.MethodPost, ts.URL+"/api/integrations", map[string]string{
		"type": "http-health", "name": "Secret Svc", "secret": "my-key",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d (%s)", resp.StatusCode, body)
	}
	var created map[string]any
	_ = json.Unmarshal(body, &created)
	if created["hasSecret"] != true {
		t.Errorf("hasSecret = %v, want true", created["hasSecret"])
	}
	// The secret value must never appear in the response.
	if bytes.Contains(body, []byte("my-key")) {
		t.Error("response leaked the secret value")
	}
}

func TestTestConnectionEndpoint(t *testing.T) {
	ts, _ := newTestServer(t, "")
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	resp, body := do(t, http.MethodPost, ts.URL+"/api/integrations/test", map[string]string{
		"type": "http-health", "baseUrl": backend.URL,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("test status = %d (%s)", resp.StatusCode, body)
	}
	var tr map[string]any
	_ = json.Unmarshal(body, &tr)
	if tr["ok"] != true {
		t.Errorf("test result ok = %v, want true (%s)", tr["ok"], body)
	}
}

func TestDiscoverySettings(t *testing.T) {
	ts, _ := newTestServer(t, "")

	// Default
	_, body := do(t, http.MethodGet, ts.URL+"/api/discovery/settings", nil)
	var ds map[string]any
	_ = json.Unmarshal(body, &ds)
	if ds["enabled"] != false || ds["mode"] != "review" || ds["available"] != false {
		t.Errorf("default settings = %v", ds)
	}

	// Update
	resp, _ := do(t, http.MethodPut, ts.URL+"/api/discovery/settings", map[string]any{
		"enabled": true, "mode": "auto",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put settings status = %d", resp.StatusCode)
	}
	_, body = do(t, http.MethodGet, ts.URL+"/api/discovery/settings", nil)
	_ = json.Unmarshal(body, &ds)
	if ds["enabled"] != true || ds["mode"] != "auto" {
		t.Errorf("after update = %v", ds)
	}

	// Invalid mode rejected
	resp, _ = do(t, http.MethodPut, ts.URL+"/api/discovery/settings", map[string]any{
		"enabled": true, "mode": "bogus",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid mode status = %d, want 400", resp.StatusCode)
	}
}

func TestResourcesEmpty(t *testing.T) {
	ts, _ := newTestServer(t, "")
	resp, body := do(t, http.MethodGet, ts.URL+"/api/resources", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("resources status = %d", resp.StatusCode)
	}
	var snap []any
	if err := json.Unmarshal(body, &snap); err != nil {
		t.Fatalf("resources not a JSON array: %s", body)
	}
}

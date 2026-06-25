package store

import (
	"path/filepath"
	"testing"
)

func openTemp(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestMigrateAndPing(t *testing.T) {
	st := openTemp(t)
	if err := st.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestEnsureActiveSeedsOnce(t *testing.T) {
	st := openTemp(t)
	def := []byte(`{"version":1}`)

	got, err := st.Scapes().EnsureActive("Default", def)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if string(got) != string(def) {
		t.Errorf("seeded spec = %s, want %s", got, def)
	}

	// Second call must not reseed; it returns the existing active spec.
	again, err := st.Scapes().EnsureActive("Default", []byte(`{"version":1,"changed":true}`))
	if err != nil {
		t.Fatalf("ensure 2: %v", err)
	}
	if string(again) != string(def) {
		t.Errorf("expected existing spec on second ensure, got %s", again)
	}
}

func TestSaveActivePersists(t *testing.T) {
	st := openTemp(t)
	if _, err := st.Scapes().EnsureActive("Default", []byte(`{"version":1}`)); err != nil {
		t.Fatal(err)
	}

	updated := []byte(`{"version":1,"meta":{"accent":"#000000"}}`)
	if err := st.Scapes().SaveActive(updated); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := st.Scapes().ActiveSpec()
	if err != nil {
		t.Fatalf("active: %v", err)
	}
	if string(got) != string(updated) {
		t.Errorf("after save = %s, want %s", got, updated)
	}
}

package secret_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/MusiThang/homescape/internal/secret"
	"github.com/MusiThang/homescape/internal/store"
)

func newStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestSealOpenRoundTrip(t *testing.T) {
	st := newStore(t)
	v := secret.New("correct horse battery staple", st.Secrets(), st.Settings())

	if !v.Enabled() {
		t.Fatal("vault should be enabled with a passphrase")
	}

	ref, err := v.Seal([]byte("my-api-key"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if ref == "" {
		t.Fatal("expected a non-empty ref")
	}

	got, err := v.Open(ref)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if string(got) != "my-api-key" {
		t.Errorf("roundtrip = %q, want my-api-key", got)
	}
}

func TestNoKeyDisabled(t *testing.T) {
	st := newStore(t)
	v := secret.New("", st.Secrets(), st.Settings())

	if v.Enabled() {
		t.Fatal("vault should be disabled without a passphrase")
	}
	if _, err := v.Seal([]byte("x")); !errors.Is(err, secret.ErrNoKey) {
		t.Errorf("seal without key: want ErrNoKey, got %v", err)
	}
	if _, err := v.Open("whatever"); !errors.Is(err, secret.ErrNoKey) {
		t.Errorf("open without key: want ErrNoKey, got %v", err)
	}
}

func TestWrongKeyFailsToOpen(t *testing.T) {
	st := newStore(t)
	// Seal with one passphrase...
	v1 := secret.New("passphrase-one", st.Secrets(), st.Settings())
	ref, err := v1.Seal([]byte("secret-value"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}

	// ...open with a different passphrase against the same store (same salt) must fail.
	v2 := secret.New("passphrase-two", st.Secrets(), st.Settings())
	if _, err := v2.Open(ref); err == nil {
		t.Error("expected decryption failure with wrong passphrase")
	}
}

func TestSaltStablePersistsAcrossVaults(t *testing.T) {
	st := newStore(t)
	v1 := secret.New("same-pass", st.Secrets(), st.Settings())
	ref, _ := v1.Seal([]byte("hello"))

	// A new vault instance with the same passphrase + same store must reuse the salt and
	// successfully decrypt.
	v2 := secret.New("same-pass", st.Secrets(), st.Settings())
	got, err := v2.Open(ref)
	if err != nil {
		t.Fatalf("open with fresh vault: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("got %q, want hello", got)
	}
}

func TestDelete(t *testing.T) {
	st := newStore(t)
	v := secret.New("pass", st.Secrets(), st.Settings())
	ref, _ := v.Seal([]byte("gone-soon"))

	if err := v.Delete(ref); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := v.Open(ref); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("open after delete: want ErrNotFound, got %v", err)
	}
}

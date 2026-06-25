package store

import (
	"errors"
	"testing"
)

func sample(id string) Integration {
	return Integration{
		ID:     id,
		Type:   "http-health",
		Name:   "Jellyfin",
		Group:  "media",
		Source: "manual",
		Status: "active",
	}
}

func TestIntegrationCRUD(t *testing.T) {
	st := openTemp(t)
	repo := st.Integrations()

	if err := repo.Create(sample("jellyfin")); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.Get("jellyfin")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Jellyfin" || got.Type != "http-health" {
		t.Errorf("unexpected integration: %+v", got)
	}

	got.Name = "Jellyfin Media"
	got.BaseURL = "http://jellyfin:8096"
	if err := repo.Update(got); err != nil {
		t.Fatalf("update: %v", err)
	}
	reread, _ := repo.Get("jellyfin")
	if reread.Name != "Jellyfin Media" || reread.BaseURL != "http://jellyfin:8096" {
		t.Errorf("update not persisted: %+v", reread)
	}

	if err := repo.Delete("jellyfin"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.Get("jellyfin"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestIntegrationListOrdered(t *testing.T) {
	st := openTemp(t)
	repo := st.Integrations()
	_ = repo.Create(Integration{ID: "b", Type: "http-health", Name: "Zeta", Source: "manual", Status: "active"})
	_ = repo.Create(Integration{ID: "a", Type: "http-health", Name: "alpha", Source: "manual", Status: "active"})

	list, err := repo.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 || list[0].Name != "alpha" || list[1].Name != "Zeta" {
		t.Errorf("expected case-insensitive name order, got %v", list)
	}
}

func TestNotFoundOnMissing(t *testing.T) {
	st := openTemp(t)
	repo := st.Integrations()
	if err := repo.Update(sample("nope")); !errors.Is(err, ErrNotFound) {
		t.Errorf("update missing: want ErrNotFound, got %v", err)
	}
	if err := repo.SetStatus("nope", "hidden"); !errors.Is(err, ErrNotFound) {
		t.Errorf("setstatus missing: want ErrNotFound, got %v", err)
	}
	if err := repo.Delete("nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete missing: want ErrNotFound, got %v", err)
	}
}

func TestDiscoverySettings(t *testing.T) {
	st := openTemp(t)
	repo := st.Integrations()

	ds, err := repo.DiscoverySettings()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if ds.Enabled || ds.Mode != "review" {
		t.Errorf("default settings = %+v, want {false review}", ds)
	}

	if err := repo.SaveDiscoverySettings(DiscoverySettings{Enabled: true, Mode: "auto"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	ds, _ = repo.DiscoverySettings()
	if !ds.Enabled || ds.Mode != "auto" {
		t.Errorf("after save = %+v, want {true auto}", ds)
	}
}

func TestDiscoveryKeyUnique(t *testing.T) {
	st := openTemp(t)
	repo := st.Integrations()
	a := sample("a")
	a.DiscoveryKey = "container-x"
	b := sample("b")
	b.DiscoveryKey = "container-x"

	if err := repo.Create(a); err != nil {
		t.Fatalf("create a: %v", err)
	}
	if err := repo.Create(b); err == nil {
		t.Error("expected unique constraint violation on duplicate discovery_key")
	}

	// But empty discovery_key (manual integrations) must allow many rows.
	if err := repo.Create(sample("m1")); err != nil {
		t.Fatalf("manual 1: %v", err)
	}
	if err := repo.Create(sample("m2")); err != nil {
		t.Fatalf("manual 2 (empty discovery_key should not collide): %v", err)
	}
}

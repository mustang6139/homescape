package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// ScapeRepo persists Scape specs and tracks which one is active. Specs are stored as raw
// validated JSON — validation is the scape package's job, not the store's.
type ScapeRepo struct {
	db *sql.DB
}

// Scapes returns a repository over this store.
func (s *Store) Scapes() *ScapeRepo { return &ScapeRepo{db: s.db} }

// EnsureActive guarantees there is an active Scape, seeding it from defaultSpecJSON on a
// fresh database. Returns the active scape's raw spec JSON.
func (r *ScapeRepo) EnsureActive(name string, defaultSpecJSON []byte) ([]byte, error) {
	raw, err := r.ActiveSpec()
	if err == nil {
		return raw, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	res, err := r.db.Exec(`INSERT INTO scapes (name, spec_json) VALUES (?, ?)`, name, string(defaultSpecJSON))
	if err != nil {
		return nil, fmt.Errorf("seed scape: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	if _, err := r.db.Exec(`INSERT INTO active_scape (id, scape_id) VALUES (1, ?)`, id); err != nil {
		return nil, fmt.Errorf("set active scape: %w", err)
	}
	return defaultSpecJSON, nil
}

// ActiveSpec returns the raw JSON of the currently active Scape.
func (r *ScapeRepo) ActiveSpec() ([]byte, error) {
	var spec string
	err := r.db.QueryRow(`
		SELECT s.spec_json
		FROM active_scape a
		JOIN scapes s ON s.id = a.scape_id
		WHERE a.id = 1`).Scan(&spec)
	if err != nil {
		return nil, err
	}
	return []byte(spec), nil
}

// SaveActive overwrites the active Scape's spec with the given validated JSON.
func (r *ScapeRepo) SaveActive(specJSON []byte) error {
	res, err := r.db.Exec(`
		UPDATE scapes
		SET spec_json = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = (SELECT scape_id FROM active_scape WHERE id = 1)`, string(specJSON))
	if err != nil {
		return fmt.Errorf("save active scape: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("no active scape to save")
	}
	return nil
}

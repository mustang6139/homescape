package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// SettingsRepo is a simple key/value store over the settings table.
type SettingsRepo struct {
	db *sql.DB
}

// Settings returns a repository over this store.
func (s *Store) Settings() *SettingsRepo { return &SettingsRepo{db: s.db} }

// Get returns the value for key and whether it was present.
func (r *SettingsRepo) Get(key string) (string, bool, error) {
	var v string
	err := r.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get setting %q: %w", key, err)
	}
	return v, true, nil
}

// Set upserts a key/value pair.
func (r *SettingsRepo) Set(key, value string) error {
	_, err := r.db.Exec(`
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}
	return nil
}

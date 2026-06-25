package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// SecretRepo stores opaque encrypted blobs. It performs no cryptography — that is the
// secret package's job; this is just durable storage keyed by ref.
type SecretRepo struct {
	db *sql.DB
}

// Secrets returns a repository over this store.
func (s *Store) Secrets() *SecretRepo { return &SecretRepo{db: s.db} }

// PutSecret upserts the nonce + ciphertext for ref.
func (r *SecretRepo) PutSecret(ref string, nonce, ciphertext []byte) error {
	_, err := r.db.Exec(`
		INSERT INTO secrets (ref, nonce, ciphertext) VALUES (?, ?, ?)
		ON CONFLICT(ref) DO UPDATE SET nonce = excluded.nonce, ciphertext = excluded.ciphertext`,
		ref, nonce, ciphertext)
	if err != nil {
		return fmt.Errorf("put secret: %w", err)
	}
	return nil
}

// GetSecret returns the stored nonce + ciphertext for ref, or ErrNotFound.
func (r *SecretRepo) GetSecret(ref string) (nonce, ciphertext []byte, err error) {
	err = r.db.QueryRow(`SELECT nonce, ciphertext FROM secrets WHERE ref = ?`, ref).Scan(&nonce, &ciphertext)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("get secret: %w", err)
	}
	return nonce, ciphertext, nil
}

// DeleteSecret removes the secret for ref (no error if absent).
func (r *SecretRepo) DeleteSecret(ref string) error {
	if _, err := r.db.Exec(`DELETE FROM secrets WHERE ref = ?`, ref); err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	return nil
}

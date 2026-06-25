package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// Integration is a registered service in the instance-local registry. The Scape spec
// references it by ID (a stable handle), so the spec never carries URLs or secrets.
type Integration struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Name         string `json:"name"`
	BaseURL      string `json:"baseUrl"`
	Group        string `json:"group"`
	Icon         string `json:"icon"`
	Source       string `json:"source"` // manual | discovery
	Status       string `json:"status"` // pending | active | hidden | stale
	SecretRef    string `json:"-"`      // never serialised to clients
	DiscoveryKey string `json:"-"`      // internal: maps to a discovered container
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// HasSecret reports whether the integration has an associated stored secret.
func (i Integration) HasSecret() bool { return i.SecretRef != "" }

// ErrNotFound is returned when an integration does not exist.
var ErrNotFound = errors.New("integration not found")

// IntegrationRepo persists the Services registry.
type IntegrationRepo struct {
	db *sql.DB
}

// Integrations returns a repository over this store.
func (s *Store) Integrations() *IntegrationRepo { return &IntegrationRepo{db: s.db} }

const integrationCols = `id, type, name, base_url, group_name, icon, source, status, secret_ref, discovery_key, created_at, updated_at`

func scanIntegration(row interface{ Scan(...any) error }) (Integration, error) {
	var it Integration
	err := row.Scan(&it.ID, &it.Type, &it.Name, &it.BaseURL, &it.Group, &it.Icon,
		&it.Source, &it.Status, &it.SecretRef, &it.DiscoveryKey, &it.CreatedAt, &it.UpdatedAt)
	return it, err
}

// List returns all integrations ordered by name.
func (r *IntegrationRepo) List() ([]Integration, error) {
	rows, err := r.db.Query(`SELECT ` + integrationCols + ` FROM integrations ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("list integrations: %w", err)
	}
	defer rows.Close()

	var out []Integration
	for rows.Next() {
		it, err := scanIntegration(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// Get returns one integration by id, or ErrNotFound.
func (r *IntegrationRepo) Get(id string) (Integration, error) {
	row := r.db.QueryRow(`SELECT `+integrationCols+` FROM integrations WHERE id = ?`, id)
	it, err := scanIntegration(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Integration{}, ErrNotFound
	}
	if err != nil {
		return Integration{}, fmt.Errorf("get integration: %w", err)
	}
	return it, nil
}

// Create inserts a new integration. created_at/updated_at default in SQL.
func (r *IntegrationRepo) Create(it Integration) error {
	_, err := r.db.Exec(`
		INSERT INTO integrations (id, type, name, base_url, group_name, icon, source, status, secret_ref, discovery_key)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		it.ID, it.Type, it.Name, it.BaseURL, it.Group, it.Icon, it.Source, it.Status, it.SecretRef, it.DiscoveryKey)
	if err != nil {
		return fmt.Errorf("create integration: %w", err)
	}
	return nil
}

// Update overwrites the mutable fields of an existing integration.
func (r *IntegrationRepo) Update(it Integration) error {
	res, err := r.db.Exec(`
		UPDATE integrations
		SET type = ?, name = ?, base_url = ?, group_name = ?, icon = ?, source = ?, status = ?, secret_ref = ?, discovery_key = ?,
		    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ?`,
		it.Type, it.Name, it.BaseURL, it.Group, it.Icon, it.Source, it.Status, it.SecretRef, it.DiscoveryKey, it.ID)
	if err != nil {
		return fmt.Errorf("update integration: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetStatus updates only the status field (used by discovery accept/hide/reconcile).
func (r *IntegrationRepo) SetStatus(id, status string) error {
	res, err := r.db.Exec(`
		UPDATE integrations SET status = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?`,
		status, id)
	if err != nil {
		return fmt.Errorf("set status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes an integration by id.
func (r *IntegrationRepo) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM integrations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete integration: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// --- discovery settings ---

// DiscoverySettings holds the auto-discovery configuration.
type DiscoverySettings struct {
	Enabled bool   `json:"enabled"`
	Mode    string `json:"mode"` // review | auto
}

// DiscoverySettings returns the current discovery settings.
func (r *IntegrationRepo) DiscoverySettings() (DiscoverySettings, error) {
	var ds DiscoverySettings
	var enabled int
	err := r.db.QueryRow(`SELECT enabled, mode FROM discovery_settings WHERE id = 1`).Scan(&enabled, &ds.Mode)
	if err != nil {
		return ds, fmt.Errorf("get discovery settings: %w", err)
	}
	ds.Enabled = enabled != 0
	return ds, nil
}

// SaveDiscoverySettings persists the discovery settings.
func (r *IntegrationRepo) SaveDiscoverySettings(ds DiscoverySettings) error {
	enabled := 0
	if ds.Enabled {
		enabled = 1
	}
	_, err := r.db.Exec(`UPDATE discovery_settings SET enabled = ?, mode = ? WHERE id = 1`, enabled, ds.Mode)
	if err != nil {
		return fmt.Errorf("save discovery settings: %w", err)
	}
	return nil
}

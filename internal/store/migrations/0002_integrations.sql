-- 0002_integrations: the instance-local Services registry, encrypted secrets, and
-- discovery settings. Integrations are referenced from the Scape spec by their id (a
-- stable handle), so the spec itself never holds URLs or secrets.

CREATE TABLE IF NOT EXISTS integrations (
    id            TEXT PRIMARY KEY,                  -- human-readable handle, e.g. "sonarr-main"
    type          TEXT NOT NULL,                     -- connector type, e.g. "http-health"
    name          TEXT NOT NULL,
    base_url      TEXT NOT NULL DEFAULT '',
    group_name    TEXT NOT NULL DEFAULT '',
    icon          TEXT NOT NULL DEFAULT '',
    source        TEXT NOT NULL DEFAULT 'manual',    -- manual | discovery
    status        TEXT NOT NULL DEFAULT 'active',     -- pending | active | hidden | stale
    secret_ref    TEXT NOT NULL DEFAULT '',           -- points at secrets.ref (empty = no secret)
    discovery_key TEXT NOT NULL DEFAULT '',           -- stable key for a discovered container
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

-- A discovered container maps to at most one integration.
CREATE UNIQUE INDEX IF NOT EXISTS idx_integrations_discovery_key
    ON integrations(discovery_key) WHERE discovery_key <> '';

CREATE TABLE IF NOT EXISTS secrets (
    ref        TEXT PRIMARY KEY,
    nonce      BLOB NOT NULL,
    ciphertext BLOB NOT NULL
);

-- Single-row settings table (id is always 1).
CREATE TABLE IF NOT EXISTS discovery_settings (
    id      INTEGER PRIMARY KEY CHECK (id = 1),
    enabled INTEGER NOT NULL DEFAULT 0,              -- 0/1
    mode    TEXT NOT NULL DEFAULT 'review'           -- review | auto
);
INSERT OR IGNORE INTO discovery_settings (id, enabled, mode) VALUES (1, 0, 'review');

-- 0001_init: instance settings, scapes, and the single active-scape pointer.

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS scapes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    spec_json  TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

-- Single-row table (id is always 1 in v1) pointing at the live scape.
CREATE TABLE IF NOT EXISTS active_scape (
    id       INTEGER PRIMARY KEY CHECK (id = 1),
    scape_id INTEGER NOT NULL REFERENCES scapes(id)
);

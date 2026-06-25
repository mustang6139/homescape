// Package scape defines the Scape spec — the declarative, portable description of a
// dashboard's state. It is the single source of truth shared (via JSON Schema) between
// the Go backend and the Svelte frontend.
package scape

// Spec is a complete dashboard state.
type Spec struct {
	Version int      `json:"version"`
	Meta    Meta     `json:"meta"`
	Layout  Layout   `json:"layout"`
	Widgets []Widget `json:"widgets"`
}

// Meta holds instance-level appearance and identity.
type Meta struct {
	Name    string `json:"name"`
	Theme   string `json:"theme"`
	Accent  string `json:"accent"`
	Density string `json:"density,omitempty"`
}

// Layout describes the grid.
type Layout struct {
	Columns int `json:"columns"`
}

// Widget is a single placed building block. Type-specific configuration lives in Options,
// kept loose for now; the L2 Composer (F4) will decompose these into primitives.
type Widget struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Column  int            `json:"column"`
	Enabled *bool          `json:"enabled,omitempty"`
	Title   string         `json:"title,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

// CurrentVersion is the spec format version this build produces.
const CurrentVersion = 1

// Default returns the seed Scape used on a fresh install (L0 "just works" baseline).
func Default() Spec {
	enabled := true
	return Spec{
		Version: CurrentVersion,
		Meta: Meta{
			Name:    "Default",
			Theme:   "midnight-glass",
			Accent:  "#E6A95C",
			Density: "cozy",
		},
		Layout: Layout{Columns: 3},
		Widgets: []Widget{
			{ID: "w-media", Type: "media-now-playing", Column: 0, Enabled: &enabled, Title: "Plex"},
			{ID: "w-stats", Type: "system-stats", Column: 1, Enabled: &enabled, Title: "System",
				Options: map[string]any{"host": "homelab-01"}},
			{ID: "w-health", Type: "service-health", Column: 1, Enabled: &enabled, Title: "Service health",
				Options: map[string]any{"services": []any{}}},
			{ID: "w-clock", Type: "clock", Column: 2, Enabled: &enabled, Title: "Clock"},
			{ID: "w-bookmarks", Type: "bookmarks", Column: 2, Enabled: &enabled, Title: "Bookmarks",
				Options: map[string]any{"links": []any{}}},
		},
	}
}

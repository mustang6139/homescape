// Package connectors is the integration layer. A Connector turns a configured service into
// normalized, typed Resources that widgets bind to — this taxonomy is the seed of the L2
// Composer (F4). Connectors never touch storage and never see encrypted data: they receive
// a Config with an already-decrypted secret and return plain values.
package connectors

import "context"

// ResourceKind enumerates the normalized data shapes connectors can provide.
type ResourceKind string

const (
	KindServiceStatus ResourceKind = "service.status"
	KindDownloadQueue ResourceKind = "download.queue" // later (qBittorrent)
	KindMediaSessions ResourceKind = "media.sessions" // later (Plex/Jellyfin)
	KindCalendarItems ResourceKind = "calendar.items" // later (*arr)
)

// Config is the connector-facing view of an integration. No storage concerns; Secret is
// already decrypted (empty when the connector needs none).
type Config struct {
	BaseURL string
	Secret  string
	Options map[string]any // per-connector knobs, for future connectors
}

// Resource is a normalized, typed piece of data keyed by kind.
type Resource struct {
	Kind ResourceKind `json:"kind"`
	Data any          `json:"data"`
}

// TestResult is the outcome of a connection test (the beginner-bridge "Test connection").
type TestResult struct {
	OK        bool   `json:"ok"`
	Version   string `json:"version,omitempty"`
	Message   string `json:"message"`
	LatencyMs int    `json:"latencyMs"`
}

// ServiceStatus is the data payload for KindServiceStatus.
type ServiceStatus struct {
	Up        bool   `json:"up"`
	LatencyMs int    `json:"latencyMs"`
	Version   string `json:"version,omitempty"`
	Message   string `json:"message,omitempty"`
}

// Connector adapts a service into one or more Resources.
type Connector interface {
	Type() string
	Provides() []ResourceKind
	Test(ctx context.Context, cfg Config) (TestResult, error)
	Fetch(ctx context.Context, cfg Config, kind ResourceKind) (Resource, error)
}

// Registry resolves connectors by type.
type Registry struct {
	byType map[string]Connector
}

// NewRegistry builds a registry from the given connectors.
func NewRegistry(cs ...Connector) *Registry {
	r := &Registry{byType: make(map[string]Connector, len(cs))}
	for _, c := range cs {
		r.byType[c.Type()] = c
	}
	return r
}

// Get returns the connector for typ, if registered.
func (r *Registry) Get(typ string) (Connector, bool) {
	c, ok := r.byType[typ]
	return c, ok
}

// Types returns the registered connector type names.
func (r *Registry) Types() []string {
	out := make([]string, 0, len(r.byType))
	for t := range r.byType {
		out = append(out, t)
	}
	return out
}

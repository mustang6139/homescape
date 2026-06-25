// Package server wires the HTTP API, the SSE live channel, and the embedded frontend.
package server

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/MusiThang/homescape/internal/connectors"
	"github.com/MusiThang/homescape/internal/discovery"
	"github.com/MusiThang/homescape/internal/secret"
	"github.com/MusiThang/homescape/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Deps bundles the server's dependencies.
type Deps struct {
	Log       *slog.Logger
	Store     *store.Store
	Vault     *secret.Vault
	Registry  *connectors.Registry
	Poller    *connectors.Poller
	Discovery *discovery.Discovery // may be nil when the Docker socket is unavailable
	Hub       *Hub
	WebFS     fs.FS // embedded built frontend (web/dist contents)
}

// Server holds dependencies shared by the HTTP handlers.
type Server struct {
	Deps
	router http.Handler
}

// New builds a Server and its router.
func New(d Deps) *Server {
	if d.Log == nil {
		d.Log = slog.Default()
	}
	s := &Server{Deps: d}
	s.router = s.routes()
	return s
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler { return s.router }

func (s *Server) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(s.logRequests)

	r.Route("/api", func(api chi.Router) {
		api.Get("/scape", s.handleGetScape)
		api.Put("/scape", s.handlePutScape)
		api.Patch("/scape", s.handlePatchScape)
		api.Get("/stats", s.handleStats)
		api.Get("/events", s.handleEvents)

		// Integrations registry
		api.Get("/integrations", s.handleListIntegrations)
		api.Post("/integrations", s.handleCreateIntegration)
		api.Post("/integrations/test", s.handleTestIntegration)
		api.Patch("/integrations/{id}", s.handleUpdateIntegration)
		api.Delete("/integrations/{id}", s.handleDeleteIntegration)

		// Live resources (initial snapshot; deltas come over SSE)
		api.Get("/resources", s.handleResources)

		// Auto-discovery
		api.Get("/discovery/pending", s.handleDiscoveryPending)
		api.Post("/discovery/{id}/accept", s.handleDiscoveryAccept)
		api.Post("/discovery/{id}/hide", s.handleDiscoveryHide)
		api.Get("/discovery/settings", s.handleGetDiscoverySettings)
		api.Put("/discovery/settings", s.handlePutDiscoverySettings)
	})

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeText(w, http.StatusOK, "ok")
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if err := s.Store.Ping(); err != nil {
			writeText(w, http.StatusServiceUnavailable, "not ready")
			return
		}
		writeText(w, http.StatusOK, "ready")
	})

	// SPA: everything else falls back to the embedded frontend.
	r.Handle("/*", s.spaHandler())
	return r
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Log.Debug("request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// refreshSoon triggers an out-of-band poll cycle so live data reflects a registry change
// promptly (rather than waiting for the next tick).
func (s *Server) refreshSoon() {
	if s.Poller != nil {
		go s.Poller.Refresh(context.Background())
	}
}

// Package server wires the HTTP API, the SSE live channel, and the embedded frontend.
package server

import (
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/MusiThang/homescape/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server holds dependencies shared by the HTTP handlers.
type Server struct {
	log    *slog.Logger
	store  *store.Store
	hub    *Hub
	webFS  fs.FS // embedded built frontend (web/dist contents)
	router http.Handler
}

// New builds a Server and its router. webFS is the embedded frontend filesystem.
func New(log *slog.Logger, st *store.Store, webFS fs.FS) *Server {
	s := &Server{
		log:   log,
		store: st,
		hub:   NewHub(),
		webFS: webFS,
	}
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
	})

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeText(w, http.StatusOK, "ok")
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if err := s.store.Ping(); err != nil {
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
		s.log.Debug("request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

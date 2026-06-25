package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/MusiThang/homescape/internal/store"
	"github.com/go-chi/chi/v5"
)

// handleResources returns the poller's current cached resources (initial load; deltas
// arrive over SSE as resource.updated events).
func (s *Server) handleResources(w http.ResponseWriter, _ *http.Request) {
	if s.Poller == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	writeJSON(w, http.StatusOK, s.Poller.Snapshot())
}

// handleDiscoveryPending lists discovered integrations awaiting review.
func (s *Server) handleDiscoveryPending(w http.ResponseWriter, _ *http.Request) {
	list, err := s.Store.Integrations().List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list integrations")
		return
	}
	out := make([]integrationView, 0)
	for _, it := range list {
		if it.Status == "pending" {
			out = append(out, view(it))
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleDiscoveryAccept(w http.ResponseWriter, r *http.Request) {
	s.setDiscoveryStatus(w, chi.URLParam(r, "id"), "active")
}

func (s *Server) handleDiscoveryHide(w http.ResponseWriter, r *http.Request) {
	s.setDiscoveryStatus(w, chi.URLParam(r, "id"), "hidden")
}

func (s *Server) setDiscoveryStatus(w http.ResponseWriter, id, status string) {
	err := s.Store.Integrations().SetStatus(id, status)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "integration not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update status")
		return
	}
	s.afterRegistryChange()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetDiscoverySettings(w http.ResponseWriter, _ *http.Request) {
	ds, err := s.Store.Integrations().DiscoverySettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load settings")
		return
	}
	// Surface whether discovery is actually wired (socket reachable) so the UI can explain
	// when the toggle has no effect.
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":   ds.Enabled,
		"mode":      ds.Mode,
		"available": s.Discovery != nil,
	})
}

func (s *Server) handlePutDiscoverySettings(w http.ResponseWriter, r *http.Request) {
	var ds store.DiscoverySettings
	if !decodeBody(w, r, &ds) {
		return
	}
	if ds.Mode != "review" && ds.Mode != "auto" {
		writeError(w, http.StatusBadRequest, "mode must be 'review' or 'auto'")
		return
	}
	if err := s.Store.Integrations().SaveDiscoverySettings(ds); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save settings")
		return
	}
	// Reconcile immediately so enabling discovery takes effect without waiting for the tick.
	if s.Discovery != nil {
		go func() { _ = s.Discovery.Reconcile(context.Background()) }()
	}
	writeJSON(w, http.StatusOK, ds)
}

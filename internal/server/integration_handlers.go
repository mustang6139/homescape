package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/MusiThang/homescape/internal/connectors"
	"github.com/MusiThang/homescape/internal/store"
	"github.com/go-chi/chi/v5"
)

const maxBodyBytes = 1 << 16 // 64 KiB for integration payloads

// integrationView is the client-facing shape of an integration (no secret material; just a
// flag indicating whether a secret is set).
type integrationView struct {
	store.Integration
	HasSecret bool `json:"hasSecret"`
}

func view(it store.Integration) integrationView {
	return integrationView{Integration: it, HasSecret: it.HasSecret()}
}

func (s *Server) handleListIntegrations(w http.ResponseWriter, _ *http.Request) {
	list, err := s.Store.Integrations().List()
	if err != nil {
		s.Log.Error("list integrations", "err", err)
		writeError(w, http.StatusInternalServerError, "could not list integrations")
		return
	}
	out := make([]integrationView, 0, len(list))
	for _, it := range list {
		out = append(out, view(it))
	}
	writeJSON(w, http.StatusOK, out)
}

type integrationInput struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
	Group   string `json:"group"`
	Icon    string `json:"icon"`
	Secret  string `json:"secret"`
}

func (s *Server) handleCreateIntegration(w http.ResponseWriter, r *http.Request) {
	var in integrationInput
	if !decodeBody(w, r, &in) {
		return
	}
	if in.Name == "" || in.Type == "" {
		writeError(w, http.StatusBadRequest, "name and type are required")
		return
	}
	if _, ok := s.Registry.Get(in.Type); !ok {
		writeError(w, http.StatusBadRequest, "unknown connector type: "+in.Type)
		return
	}

	secretRef, ok := s.sealSecret(w, in.Secret)
	if !ok {
		return
	}

	id, err := s.Store.Integrations().GenerateID(in.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate id")
		return
	}
	it := store.Integration{
		ID: id, Type: in.Type, Name: in.Name, BaseURL: in.BaseURL,
		Group: in.Group, Icon: in.Icon, Source: "manual", Status: "active",
		SecretRef: secretRef,
	}
	if err := s.Store.Integrations().Create(it); err != nil {
		s.Log.Error("create integration", "err", err)
		writeError(w, http.StatusInternalServerError, "could not create integration")
		return
	}
	s.afterRegistryChange()
	writeJSON(w, http.StatusCreated, view(it))
}

type integrationPatch struct {
	Name    *string `json:"name"`
	BaseURL *string `json:"baseUrl"`
	Group   *string `json:"group"`
	Icon    *string `json:"icon"`
	Status  *string `json:"status"`
	Secret  *string `json:"secret"`
}

func (s *Server) handleUpdateIntegration(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	it, err := s.Store.Integrations().Get(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "integration not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load integration")
		return
	}

	var p integrationPatch
	if !decodeBody(w, r, &p) {
		return
	}
	if p.Name != nil {
		it.Name = *p.Name
	}
	if p.BaseURL != nil {
		it.BaseURL = *p.BaseURL
	}
	if p.Group != nil {
		it.Group = *p.Group
	}
	if p.Icon != nil {
		it.Icon = *p.Icon
	}
	if p.Status != nil {
		it.Status = *p.Status
	}
	if p.Secret != nil {
		ref, ok := s.sealSecret(w, *p.Secret)
		if !ok {
			return
		}
		// Replace the old secret if there was one.
		if it.SecretRef != "" {
			_ = s.Vault.Delete(it.SecretRef)
		}
		it.SecretRef = ref
	}

	if err := s.Store.Integrations().Update(it); err != nil {
		s.Log.Error("update integration", "err", err)
		writeError(w, http.StatusInternalServerError, "could not update integration")
		return
	}
	s.afterRegistryChange()
	writeJSON(w, http.StatusOK, view(it))
}

func (s *Server) handleDeleteIntegration(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	it, err := s.Store.Integrations().Get(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "integration not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load integration")
		return
	}
	if it.SecretRef != "" {
		_ = s.Vault.Delete(it.SecretRef)
	}
	if err := s.Store.Integrations().Delete(id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete integration")
		return
	}
	s.afterRegistryChange()
	w.WriteHeader(http.StatusNoContent)
}

// handleTestIntegration runs an ad-hoc connection test using the posted config — so a user
// can verify a service before saving it (the beginner-bridge "Test connection").
func (s *Server) handleTestIntegration(w http.ResponseWriter, r *http.Request) {
	var in integrationInput
	if !decodeBody(w, r, &in) {
		return
	}
	conn, ok := s.Registry.Get(in.Type)
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown connector type: "+in.Type)
		return
	}
	cfg := connectors.Config{BaseURL: in.BaseURL, Secret: in.Secret}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	res, err := conn.Test(ctx, cfg)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// sealSecret encrypts plaintext (if any). Returns the ref and whether to proceed; on a
// configuration error it writes the response and returns ok=false.
func (s *Server) sealSecret(w http.ResponseWriter, plaintext string) (string, bool) {
	if plaintext == "" {
		return "", true
	}
	if !s.Vault.Enabled() {
		writeError(w, http.StatusBadRequest, "set HS_SECRET_KEY to store credentials")
		return "", false
	}
	ref, err := s.Vault.Seal([]byte(plaintext))
	if err != nil {
		s.Log.Error("seal secret", "err", err)
		writeError(w, http.StatusInternalServerError, "could not store secret")
		return "", false
	}
	return ref, true
}

// afterRegistryChange refreshes live data and notifies clients that the registry changed.
func (s *Server) afterRegistryChange() {
	s.refreshSoon()
	s.Hub.Broadcast(Event{Type: "integrations.changed"})
}

func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read body")
		return false
	}
	if err := json.Unmarshal(body, v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return false
	}
	return true
}

package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/MusiThang/homescape/internal/scape"
)

const maxSpecBytes = 1 << 20 // 1 MiB is plenty for a spec

// handleGetScape returns the active Scape spec as raw JSON.
func (s *Server) handleGetScape(w http.ResponseWriter, _ *http.Request) {
	raw, err := s.store.Scapes().ActiveSpec()
	if err != nil {
		s.log.Error("get active scape", "err", err)
		writeError(w, http.StatusInternalServerError, "could not load scape")
		return
	}
	writeRawJSON(w, http.StatusOK, raw)
}

// handlePutScape replaces the entire active spec (validate → persist → broadcast).
func (s *Server) handlePutScape(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxSpecBytes))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read body")
		return
	}
	if err := scape.Validate(body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := s.store.Scapes().SaveActive(body); err != nil {
		s.log.Error("save scape", "err", err)
		writeError(w, http.StatusInternalServerError, "could not save scape")
		return
	}
	s.hub.Broadcast(Event{Type: "scape.updated", Data: json.RawMessage(body)})
	writeRawJSON(w, http.StatusOK, body)
}

// handlePatchScape deep-merges a partial spec into the active one (the "live" experience:
// e.g. changing only the accent colour). The merged result is re-validated before saving.
func (s *Server) handlePatchScape(w http.ResponseWriter, r *http.Request) {
	patchBody, err := io.ReadAll(io.LimitReader(r.Body, maxSpecBytes))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read body")
		return
	}
	var patch map[string]any
	if err := json.Unmarshal(patchBody, &patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON patch")
		return
	}

	currentRaw, err := s.store.Scapes().ActiveSpec()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load scape")
		return
	}
	var current map[string]any
	if err := json.Unmarshal(currentRaw, &current); err != nil {
		writeError(w, http.StatusInternalServerError, "stored spec is corrupt")
		return
	}

	merged := deepMerge(current, patch)
	mergedRaw, err := json.Marshal(merged)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not encode merged spec")
		return
	}
	if err := scape.Validate(mergedRaw); err != nil {
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("patch produced invalid spec: %v", err))
		return
	}
	if err := s.store.Scapes().SaveActive(mergedRaw); err != nil {
		s.log.Error("save scape", "err", err)
		writeError(w, http.StatusInternalServerError, "could not save scape")
		return
	}
	s.hub.Broadcast(Event{Type: "scape.updated", Data: json.RawMessage(mergedRaw)})
	writeRawJSON(w, http.StatusOK, mergedRaw)
}

// deepMerge recursively merges src into dst for nested JSON objects. Non-object values
// (including arrays) are replaced wholesale — arrays are treated as atomic.
func deepMerge(dst, src map[string]any) map[string]any {
	out := make(map[string]any, len(dst))
	for k, v := range dst {
		out[k] = v
	}
	for k, sv := range src {
		if sm, ok := sv.(map[string]any); ok {
			if dm, ok := out[k].(map[string]any); ok {
				out[k] = deepMerge(dm, sm)
				continue
			}
		}
		out[k] = sv
	}
	return out
}

// handleEvents is the SSE endpoint clients subscribe to for live updates.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := s.hub.subscribe()
	defer s.hub.unsubscribe(ch)

	// Initial comment so the client knows the stream is open.
	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case payload, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		}
	}
}

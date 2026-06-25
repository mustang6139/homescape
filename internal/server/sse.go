package server

import (
	"encoding/json"
	"sync"
)

// Event is a server-sent event pushed to subscribed clients.
type Event struct {
	Type string `json:"type"` // e.g. "scape.updated"
	Data any    `json:"data,omitempty"`
}

// Hub is a tiny fan-out broadcaster for SSE clients.
type Hub struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

// NewHub creates an empty Hub.
func NewHub() *Hub {
	return &Hub{clients: make(map[chan []byte]struct{})}
}

// subscribe registers a new client channel.
func (h *Hub) subscribe() chan []byte {
	ch := make(chan []byte, 8)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// unsubscribe removes and closes a client channel.
func (h *Hub) unsubscribe(ch chan []byte) {
	h.mu.Lock()
	if _, ok := h.clients[ch]; ok {
		delete(h.clients, ch)
		close(ch)
	}
	h.mu.Unlock()
}

// Broadcast serialises an event and sends it to all connected clients. Slow clients that
// can't keep up drop the message rather than block the broadcaster.
func (h *Hub) Broadcast(ev Event) {
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- payload:
		default:
		}
	}
}

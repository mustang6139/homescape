package connectors

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"
	"sync"
	"time"
)

// ResourceUpdate is the payload broadcast (and cached) when a resource changes.
type ResourceUpdate struct {
	IntegrationID string       `json:"integrationId"`
	Kind          ResourceKind `json:"kind"`
	Data          any          `json:"data"`
	At            string       `json:"at"`
}

// Target is one active integration the poller should fetch, with a ready-to-use Config
// (secret already decrypted).
type Target struct {
	ID     string
	Type   string
	Config Config
}

// Source supplies the current set of active targets. Abstracted so the poller is testable
// without the store/secret packages.
type Source interface {
	ActiveTargets() ([]Target, error)
}

const maxConcurrency = 8

// Poller periodically fetches each active target's resources, caches the latest, and emits
// an update only when the value changes. The frontend never calls services directly — this
// is the single place outbound requests happen.
type Poller struct {
	reg      *Registry
	src      Source
	interval time.Duration
	emit     func(ResourceUpdate)
	log      *slog.Logger

	mu    sync.RWMutex
	cache map[string]ResourceUpdate // key: integrationID + "|" + kind
	last  map[string]string         // key -> last JSON, for change detection
}

// NewPoller builds a poller. emit may be nil (e.g. in tests).
func NewPoller(reg *Registry, src Source, interval time.Duration, emit func(ResourceUpdate), log *slog.Logger) *Poller {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if log == nil {
		log = slog.Default()
	}
	return &Poller{
		reg:      reg,
		src:      src,
		interval: interval,
		emit:     emit,
		log:      log,
		cache:    make(map[string]ResourceUpdate),
		last:     make(map[string]string),
	}
}

func key(id string, kind ResourceKind) string { return id + "|" + string(kind) }

// Run polls immediately, then on every interval, until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) {
	p.cycle(ctx)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.cycle(ctx)
		}
	}
}

// cycle fetches all current targets concurrently and prunes cache for removed targets.
func (p *Poller) cycle(ctx context.Context) {
	targets, err := p.src.ActiveTargets()
	if err != nil {
		p.log.Error("poller: list targets", "err", err)
		return
	}

	live := make(map[string]struct{})
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for _, t := range targets {
		conn, ok := p.reg.Get(t.Type)
		if !ok {
			continue
		}
		for _, kind := range conn.Provides() {
			live[key(t.ID, kind)] = struct{}{}
			wg.Add(1)
			sem <- struct{}{}
			go func(t Target, conn Connector, kind ResourceKind) {
				defer wg.Done()
				defer func() { <-sem }()
				p.pollOne(ctx, t, conn, kind)
			}(t, conn, kind)
		}
	}
	wg.Wait()
	p.prune(live)
}

func (p *Poller) pollOne(ctx context.Context, t Target, conn Connector, kind ResourceKind) {
	// Small jitter to avoid hammering many services at the same instant.
	jitter := time.Duration(rand.Intn(750)) * time.Millisecond
	select {
	case <-ctx.Done():
		return
	case <-time.After(jitter):
	}

	res, err := conn.Fetch(ctx, t.Config, kind)
	if err != nil {
		p.log.Debug("poller: fetch", "integration", t.ID, "kind", kind, "err", err)
		return
	}

	encoded, _ := json.Marshal(res.Data)
	k := key(t.ID, kind)

	p.mu.Lock()
	changed := p.last[k] != string(encoded)
	update := ResourceUpdate{
		IntegrationID: t.ID,
		Kind:          kind,
		Data:          res.Data,
		At:            time.Now().UTC().Format(time.RFC3339),
	}
	p.cache[k] = update
	p.last[k] = string(encoded)
	p.mu.Unlock()

	if changed && p.emit != nil {
		p.emit(update)
	}
}

// prune drops cache entries whose target/kind is no longer active.
func (p *Poller) prune(live map[string]struct{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for k := range p.cache {
		if _, ok := live[k]; !ok {
			delete(p.cache, k)
			delete(p.last, k)
		}
	}
}

// Snapshot returns the current cached resources (for the initial REST load).
func (p *Poller) Snapshot() []ResourceUpdate {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]ResourceUpdate, 0, len(p.cache))
	for _, u := range p.cache {
		out = append(out, u)
	}
	return out
}

// Refresh triggers an immediate poll cycle (e.g. after a registry change).
func (p *Poller) Refresh(ctx context.Context) { p.cycle(ctx) }

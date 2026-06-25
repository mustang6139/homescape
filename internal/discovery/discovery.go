package discovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/MusiThang/homescape/internal/store"
)

// Registry is the subset of the integration store discovery needs (interface for testing).
type Registry interface {
	List() ([]store.Integration, error)
	Get(id string) (store.Integration, error)
	Create(it store.Integration) error
	Update(it store.Integration) error
	SetStatus(id, status string) error
	DiscoverySettings() (store.DiscoverySettings, error)
}

// Discovery reconciles labelled containers into the Services registry and exposes live
// container state for the docker-status connector.
type Discovery struct {
	client   DockerClient
	repo     Registry
	log      *slog.Logger
	onChange func()

	mu      sync.RWMutex
	running map[string]bool // discovery_key -> running
}

// New builds a Discovery service.
func New(client DockerClient, repo Registry, log *slog.Logger) *Discovery {
	if log == nil {
		log = slog.Default()
	}
	return &Discovery{
		client:  client,
		repo:    repo,
		log:     log,
		running: make(map[string]bool),
	}
}

// SetOnChange registers a callback fired after a reconcile changes the registry (used to
// refresh the poller).
func (d *Discovery) SetOnChange(f func()) { d.onChange = f }

// Running implements connectors.StateProvider.
func (d *Discovery) Running(key string) (bool, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	v, ok := d.running[key]
	return v, ok
}

// Run does an initial reconcile, then reconciles on container events (debounced) and on a
// periodic safety-net tick, until ctx is cancelled.
func (d *Discovery) Run(ctx context.Context) {
	d.reconcileLogged(ctx)

	events := d.client.Events(ctx)
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// Debounce bursts of events (e.g. `compose up`) into a single reconcile.
	var debounce *time.Timer
	debounced := make(chan struct{}, 1)

	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			if debounce == nil {
				debounce = time.AfterFunc(750*time.Millisecond, func() {
					select {
					case debounced <- struct{}{}:
					default:
					}
				})
			} else {
				debounce.Reset(750 * time.Millisecond)
			}
		case <-debounced:
			d.reconcileLogged(ctx)
		case <-ticker.C:
			d.reconcileLogged(ctx)
		}
	}
}

func (d *Discovery) reconcileLogged(ctx context.Context) {
	if err := d.Reconcile(ctx); err != nil {
		d.log.Error("discovery: reconcile", "err", err)
	}
}

// Reconcile syncs the current container set into the registry. No-op when discovery is
// disabled in settings.
func (d *Discovery) Reconcile(ctx context.Context) error {
	settings, err := d.repo.DiscoverySettings()
	if err != nil {
		return err
	}
	if !settings.Enabled {
		return nil
	}

	containers, err := d.client.ListContainers(ctx)
	if err != nil {
		return err
	}

	// Build the desired set from candidate containers, and refresh running state.
	desired := make(map[string]desiredIntegration)
	running := make(map[string]bool)
	for _, c := range containers {
		if !isCandidate(c) {
			continue
		}
		key := discoveryKey(c)
		desired[key] = buildDesired(key, c)
		running[key] = c.Running()
	}

	d.mu.Lock()
	d.running = running
	d.mu.Unlock()

	existing, err := d.repo.List()
	if err != nil {
		return err
	}
	existingByKey := make(map[string]store.Integration)
	for _, it := range existing {
		if it.Source == "discovery" && it.DiscoveryKey != "" {
			existingByKey[it.DiscoveryKey] = it
		}
	}

	changed := false

	// Upsert desired integrations.
	for key, d2 := range desired {
		if ex, ok := existingByKey[key]; ok {
			if d.applyUpdate(ex, d2) {
				changed = true
			}
		} else {
			status := "pending"
			if settings.Mode == "auto" {
				status = "active"
			}
			id := d.uniqueID(d2.name)
			it := store.Integration{
				ID: id, Type: d2.connType, Name: d2.name, BaseURL: d2.baseURL,
				Group: d2.group, Icon: d2.icon, Source: "discovery", Status: status,
				DiscoveryKey: key,
			}
			if err := d.repo.Create(it); err != nil {
				d.log.Error("discovery: create integration", "id", id, "err", err)
				continue
			}
			changed = true
		}
	}

	// Mark integrations whose container disappeared as stale (never delete — config restores
	// when the container returns).
	for key, ex := range existingByKey {
		if _, ok := desired[key]; !ok && ex.Status != "stale" {
			if err := d.repo.SetStatus(ex.ID, "stale"); err != nil {
				d.log.Error("discovery: mark stale", "id", ex.ID, "err", err)
				continue
			}
			changed = true
		}
	}

	if changed && d.onChange != nil {
		d.onChange()
	}
	return nil
}

// applyUpdate updates an existing discovered integration's mutable fields and restores it
// from stale. Returns whether anything changed.
func (d *Discovery) applyUpdate(ex store.Integration, want desiredIntegration) bool {
	next := ex
	next.Type = want.connType
	next.Name = want.name
	next.BaseURL = want.baseURL
	next.Group = want.group
	next.Icon = want.icon
	if ex.Status == "stale" {
		next.Status = "active"
	}
	if next == ex {
		return false
	}
	if err := d.repo.Update(next); err != nil {
		d.log.Error("discovery: update integration", "id", ex.ID, "err", err)
		return false
	}
	return true
}

type desiredIntegration struct {
	key      string
	connType string
	name     string
	baseURL  string
	group    string
	icon     string
}

func buildDesired(key string, c Container) desiredIntegration {
	name := c.Labels["homescape.name"]
	if name == "" {
		name = c.Name()
	}
	url := c.Labels["homescape.url"]
	connType := "docker-status"
	if url != "" {
		connType = "http-health"
	}
	return desiredIntegration{
		key:      key,
		connType: connType,
		name:     name,
		baseURL:  url,
		group:    c.Labels["homescape.group"],
		icon:     c.Labels["homescape.icon"],
	}
}

// discoveryKey resolves a stable identity: explicit homescape.id, else compose
// project+service, else container name.
func discoveryKey(c Container) string {
	if v := c.Labels["homescape.id"]; v != "" {
		return "id:" + v
	}
	proj := c.Labels["com.docker.compose.project"]
	svc := c.Labels["com.docker.compose.service"]
	if proj != "" && svc != "" {
		return "compose:" + proj + "/" + svc
	}
	return "name:" + c.Name()
}

func isCandidate(c Container) bool {
	switch strings.ToLower(strings.TrimSpace(c.Labels["homescape.enable"])) {
	case "true", "1", "yes", "on":
		return true
	}
	return false
}

// uniqueID generates a handle from name, appending a counter on collision.
func (d *Discovery) uniqueID(name string) string {
	base := slug(name)
	if base == "" {
		base = "service"
	}
	candidate := base
	for i := 2; ; i++ {
		_, err := d.repo.Get(candidate)
		if errors.Is(err, store.ErrNotFound) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

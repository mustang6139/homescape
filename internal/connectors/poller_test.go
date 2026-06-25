package connectors

import (
	"context"
	"sync"
	"testing"
)

// fakeSource returns a controllable set of targets.
type fakeSource struct {
	mu      sync.Mutex
	targets []Target
}

func (f *fakeSource) set(ts ...Target) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.targets = ts
}

func (f *fakeSource) ActiveTargets() ([]Target, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Target(nil), f.targets...), nil
}

// fakeConnector returns a status driven by a shared pointer so tests can flip it.
type fakeConnector struct {
	mu sync.Mutex
	up bool
}

func (c *fakeConnector) Type() string             { return "fake" }
func (c *fakeConnector) Provides() []ResourceKind { return []ResourceKind{KindServiceStatus} }
func (c *fakeConnector) Test(context.Context, Config) (TestResult, error) {
	return TestResult{OK: true}, nil
}
func (c *fakeConnector) Fetch(_ context.Context, _ Config, kind ResourceKind) (Resource, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return Resource{Kind: kind, Data: ServiceStatus{Up: c.up}}, nil
}
func (c *fakeConnector) setUp(v bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.up = v
}

func TestPollerCachesAndEmitsOnChange(t *testing.T) {
	conn := &fakeConnector{up: true}
	src := &fakeSource{}
	src.set(Target{ID: "svc1", Type: "fake"})

	var mu sync.Mutex
	var emitted []ResourceUpdate
	emit := func(u ResourceUpdate) {
		mu.Lock()
		emitted = append(emitted, u)
		mu.Unlock()
	}

	p := NewPoller(NewRegistry(conn), src, 0, emit, nil)
	ctx := context.Background()

	// First cycle: new value → emitted + cached.
	p.cycle(ctx)
	if got := len(p.Snapshot()); got != 1 {
		t.Fatalf("snapshot len = %d, want 1", got)
	}
	mu.Lock()
	if len(emitted) != 1 {
		t.Errorf("emit count after first cycle = %d, want 1", len(emitted))
	}
	mu.Unlock()

	// Second cycle, unchanged: no new emit.
	p.cycle(ctx)
	mu.Lock()
	if len(emitted) != 1 {
		t.Errorf("emit count after unchanged cycle = %d, want 1", len(emitted))
	}
	mu.Unlock()

	// Flip the status: should emit again.
	conn.setUp(false)
	p.cycle(ctx)
	mu.Lock()
	if len(emitted) != 2 {
		t.Errorf("emit count after change = %d, want 2", len(emitted))
	}
	mu.Unlock()
}

func TestPollerPrunesRemovedTargets(t *testing.T) {
	conn := &fakeConnector{up: true}
	src := &fakeSource{}
	src.set(Target{ID: "svc1", Type: "fake"}, Target{ID: "svc2", Type: "fake"})

	p := NewPoller(NewRegistry(conn), src, 0, nil, nil)
	ctx := context.Background()

	p.cycle(ctx)
	if got := len(p.Snapshot()); got != 2 {
		t.Fatalf("snapshot len = %d, want 2", got)
	}

	// Remove one target; its cache entry must be pruned.
	src.set(Target{ID: "svc1", Type: "fake"})
	p.cycle(ctx)
	snap := p.Snapshot()
	if len(snap) != 1 || snap[0].IntegrationID != "svc1" {
		t.Errorf("after prune snapshot = %+v, want only svc1", snap)
	}
}

func TestPollerSkipsUnknownType(t *testing.T) {
	src := &fakeSource{}
	src.set(Target{ID: "svc1", Type: "nonexistent"})
	p := NewPoller(NewRegistry(&fakeConnector{}), src, 0, nil, nil)

	p.cycle(context.Background())
	if got := len(p.Snapshot()); got != 0 {
		t.Errorf("snapshot len = %d, want 0 (unknown type skipped)", got)
	}
}

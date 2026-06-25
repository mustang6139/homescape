package discovery

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/MusiThang/homescape/internal/store"
)

type fakeDocker struct {
	containers []Container
}

func (f *fakeDocker) ListContainers(context.Context) ([]Container, error) { return f.containers, nil }
func (f *fakeDocker) Events(context.Context) <-chan struct{}              { return make(chan struct{}) }
func (f *fakeDocker) Ping(context.Context) error                          { return nil }

func cont(name string, labels map[string]string, state string) Container {
	return Container{ID: name, Names: []string{"/" + name}, Labels: labels, State: state}
}

func newRepo(t *testing.T) *store.IntegrationRepo {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st.Integrations()
}

func enable(t *testing.T, repo *store.IntegrationRepo, mode string) {
	t.Helper()
	if err := repo.SaveDiscoverySettings(store.DiscoverySettings{Enabled: true, Mode: mode}); err != nil {
		t.Fatalf("enable: %v", err)
	}
}

func TestReconcileDisabledNoop(t *testing.T) {
	repo := newRepo(t)
	d := New(&fakeDocker{containers: []Container{
		cont("jellyfin", map[string]string{"homescape.enable": "true"}, "running"),
	}}, repo, nil)

	if err := d.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	list, _ := repo.List()
	if len(list) != 0 {
		t.Errorf("disabled discovery should create nothing, got %d", len(list))
	}
}

func TestReconcileCreatesPending(t *testing.T) {
	repo := newRepo(t)
	enable(t, repo, "review")
	d := New(&fakeDocker{containers: []Container{
		cont("jellyfin", map[string]string{
			"homescape.enable": "true",
			"homescape.name":   "Jellyfin",
			"homescape.group":  "media",
		}, "running"),
		cont("ignored", map[string]string{"foo": "bar"}, "running"), // no enable label
	}}, repo, nil)

	if err := d.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	list, _ := repo.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 discovered integration, got %d", len(list))
	}
	it := list[0]
	if it.Status != "pending" {
		t.Errorf("review mode should create pending, got %q", it.Status)
	}
	if it.Type != "docker-status" {
		t.Errorf("no url → docker-status, got %q", it.Type)
	}
	if it.Name != "Jellyfin" || it.Group != "media" || it.Source != "discovery" {
		t.Errorf("unexpected fields: %+v", it)
	}
	if it.DiscoveryKey != "name:jellyfin" {
		t.Errorf("discovery key = %q, want name:jellyfin", it.DiscoveryKey)
	}
}

func TestReconcileAutoModeAndURL(t *testing.T) {
	repo := newRepo(t)
	enable(t, repo, "auto")
	d := New(&fakeDocker{containers: []Container{
		cont("sonarr", map[string]string{
			"homescape.enable": "true",
			"homescape.url":    "http://sonarr:8989",
		}, "running"),
	}}, repo, nil)

	_ = d.Reconcile(context.Background())
	list, _ := repo.List()
	if len(list) != 1 || list[0].Status != "active" {
		t.Fatalf("auto mode should create active, got %+v", list)
	}
	if list[0].Type != "http-health" || list[0].BaseURL != "http://sonarr:8989" {
		t.Errorf("url present → http-health with base url, got %+v", list[0])
	}
}

func TestStaleAndReappear(t *testing.T) {
	repo := newRepo(t)
	enable(t, repo, "auto")
	c := cont("grafana", map[string]string{"homescape.enable": "true"}, "running")
	fd := &fakeDocker{containers: []Container{c}}
	d := New(fd, repo, nil)

	_ = d.Reconcile(context.Background())
	list, _ := repo.List()
	id := list[0].ID

	// Container disappears → stale, not deleted.
	fd.containers = nil
	_ = d.Reconcile(context.Background())
	it, _ := repo.Get(id)
	if it.Status != "stale" {
		t.Errorf("missing container should be stale, got %q", it.Status)
	}

	// Container returns → restored to active, same id (stable key).
	fd.containers = []Container{c}
	_ = d.Reconcile(context.Background())
	it, _ = repo.Get(id)
	if it.Status != "active" {
		t.Errorf("returned container should be active, got %q", it.Status)
	}
	if all, _ := repo.List(); len(all) != 1 {
		t.Errorf("reappearance must reuse the integration, got %d rows", len(all))
	}
}

func TestRunningState(t *testing.T) {
	repo := newRepo(t)
	enable(t, repo, "auto")
	d := New(&fakeDocker{containers: []Container{
		cont("plex", map[string]string{"homescape.enable": "true"}, "running"),
		cont("radarr", map[string]string{"homescape.enable": "true"}, "exited"),
	}}, repo, nil)
	_ = d.Reconcile(context.Background())

	if up, known := d.Running("name:plex"); !known || !up {
		t.Errorf("plex should be running")
	}
	if up, known := d.Running("name:radarr"); !known || up {
		t.Errorf("radarr should be known but not running")
	}
	if _, known := d.Running("name:nope"); known {
		t.Errorf("unknown key should not be known")
	}
}

func TestDiscoveryKeyLayering(t *testing.T) {
	id := discoveryKey(cont("x", map[string]string{"homescape.id": "myid"}, "running"))
	if id != "id:myid" {
		t.Errorf("explicit id: got %q", id)
	}
	comp := discoveryKey(cont("x", map[string]string{
		"com.docker.compose.project": "media",
		"com.docker.compose.service": "sonarr",
	}, "running"))
	if comp != "compose:media/sonarr" {
		t.Errorf("compose: got %q", comp)
	}
	if name := discoveryKey(cont("plex", nil, "running")); name != "name:plex" {
		t.Errorf("name fallback: got %q", name)
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Jellyfin":     "jellyfin",
		"My Cool App!": "my-cool-app",
		"  spaced  ":   "spaced",
		"a__b--c":      "a-b-c",
		"###":          "",
	}
	for in, want := range cases {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUniqueID(t *testing.T) {
	repo := newRepo(t)
	d := New(&fakeDocker{}, repo, nil)
	_ = repo.Create(store.Integration{ID: "jellyfin", Type: "http-health", Name: "x", Source: "manual", Status: "active"})

	if got := d.uniqueID("Jellyfin"); got != "jellyfin-2" {
		t.Errorf("collision should append counter, got %q", got)
	}
}

package connectors

import (
	"context"
	"testing"
)

type fakeState struct {
	running map[string]bool
}

func (f fakeState) Running(key string) (bool, bool) {
	v, ok := f.running[key]
	return v, ok
}

func dockerStatusOf(t *testing.T, ds *DockerStatus, key string) ServiceStatus {
	t.Helper()
	res, err := ds.Fetch(context.Background(), Config{Options: map[string]any{"key": key}}, KindServiceStatus)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	return res.Data.(ServiceStatus)
}

func TestDockerStatus(t *testing.T) {
	ds := NewDockerStatus(fakeState{running: map[string]bool{
		"name:up":   true,
		"name:down": false,
	}})

	if s := dockerStatusOf(t, ds, "name:up"); !s.Up || s.Message != "running" {
		t.Errorf("running container: got %+v", s)
	}
	if s := dockerStatusOf(t, ds, "name:down"); s.Up || s.Message != "stopped" {
		t.Errorf("stopped container: got %+v", s)
	}
	if s := dockerStatusOf(t, ds, "name:unknown"); s.Up || s.Message != "container not found" {
		t.Errorf("unknown container: got %+v", s)
	}
}

func TestDockerStatusNoKey(t *testing.T) {
	ds := NewDockerStatus(fakeState{})
	res, _ := ds.Fetch(context.Background(), Config{}, KindServiceStatus)
	if s := res.Data.(ServiceStatus); s.Up || s.Message != "no discovery key" {
		t.Errorf("missing key: got %+v", s)
	}
}

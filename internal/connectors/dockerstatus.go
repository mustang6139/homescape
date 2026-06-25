package connectors

import (
	"context"
	"fmt"
)

// StateProvider exposes live container running-state, keyed by an integration's
// discovery_key. Implemented by the discovery subsystem.
type StateProvider interface {
	Running(key string) (up bool, known bool)
}

// DockerStatus reports service.status from a discovered container's Docker state. It is the
// second status source (alongside http-health): discovered containers without a homescape.url
// label get this connector, so a service is "up" simply by running — no URL needed.
type DockerStatus struct {
	sp StateProvider
}

// NewDockerStatus builds the connector backed by a StateProvider.
func NewDockerStatus(sp StateProvider) *DockerStatus { return &DockerStatus{sp: sp} }

func (d *DockerStatus) Type() string { return "docker-status" }

func (d *DockerStatus) Provides() []ResourceKind { return []ResourceKind{KindServiceStatus} }

func (d *DockerStatus) status(cfg Config) ServiceStatus {
	key, _ := cfg.Options["key"].(string)
	if key == "" {
		return ServiceStatus{Up: false, Message: "no discovery key"}
	}
	up, known := d.sp.Running(key)
	if !known {
		return ServiceStatus{Up: false, Message: "container not found"}
	}
	if up {
		return ServiceStatus{Up: true, Message: "running"}
	}
	return ServiceStatus{Up: false, Message: "stopped"}
}

func (d *DockerStatus) Test(_ context.Context, cfg Config) (TestResult, error) {
	s := d.status(cfg)
	return TestResult{OK: s.Up, Message: s.Message}, nil
}

func (d *DockerStatus) Fetch(_ context.Context, cfg Config, kind ResourceKind) (Resource, error) {
	if kind != KindServiceStatus {
		return Resource{}, fmt.Errorf("docker-status does not provide %q", kind)
	}
	return Resource{Kind: KindServiceStatus, Data: d.status(cfg)}, nil
}

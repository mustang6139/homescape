// Package discovery watches the Docker socket (opt-in, read-only) and reconciles labelled
// containers into the Services registry. It uses a minimal direct HTTP client over the unix
// socket rather than the full Docker SDK, keeping the dependency tree and image small.
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Container is the subset of Docker container info discovery needs.
type Container struct {
	ID     string
	Names  []string
	Labels map[string]string
	State  string
}

// Name returns the primary container name without the leading slash.
func (c Container) Name() string {
	if len(c.Names) == 0 {
		return ""
	}
	return strings.TrimPrefix(c.Names[0], "/")
}

// Running reports whether the container is currently running.
func (c Container) Running() bool { return c.State == "running" }

// DockerClient is the minimal interface discovery depends on (so it is testable).
type DockerClient interface {
	ListContainers(ctx context.Context) ([]Container, error)
	// Events returns a channel that signals on any container event; it closes when ctx ends.
	Events(ctx context.Context) <-chan struct{}
	Ping(ctx context.Context) error
}

// socketClient talks to the Docker Engine API over a unix socket.
type socketClient struct {
	http *http.Client
}

// Dial creates a Docker client for the given unix socket path.
func Dial(socketPath string) *socketClient {
	tr := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socketPath)
		},
	}
	return &socketClient{http: &http.Client{Transport: tr, Timeout: 10 * time.Second}}
}

const dockerHost = "http://docker"

func (c *socketClient) Ping(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, dockerHost+"/_ping", nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("docker ping: status %d", resp.StatusCode)
	}
	return nil
}

func (c *socketClient) ListContainers(ctx context.Context) ([]Container, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, dockerHost+"/containers/json?all=1", nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list containers: status %d", resp.StatusCode)
	}

	var raw []struct {
		ID     string            `json:"Id"`
		Names  []string          `json:"Names"`
		Labels map[string]string `json:"Labels"`
		State  string            `json:"State"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode containers: %w", err)
	}
	out := make([]Container, len(raw))
	for i, r := range raw {
		out[i] = Container{ID: r.ID, Names: r.Names, Labels: r.Labels, State: r.State}
	}
	return out, nil
}

// Events streams container events as signals. It reconnects on stream errors while ctx is
// alive, and closes the returned channel when ctx ends.
func (c *socketClient) Events(ctx context.Context) <-chan struct{} {
	out := make(chan struct{}, 1)
	go func() {
		defer close(out)
		for ctx.Err() == nil {
			c.streamEvents(ctx, out)
			// Brief pause before reconnecting (e.g. daemon restart).
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}()
	return out
}

func (c *socketClient) streamEvents(ctx context.Context, out chan<- struct{}) {
	q := url.Values{}
	q.Set("filters", `{"type":["container"]}`)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, dockerHost+"/events?"+q.Encode(), nil)

	resp, err := c.http.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	for ctx.Err() == nil {
		var ev map[string]any
		if err := dec.Decode(&ev); err != nil {
			return // stream ended or errored; caller reconnects
		}
		select {
		case out <- struct{}{}:
		default: // a signal is already pending; coalesce
		}
	}
}

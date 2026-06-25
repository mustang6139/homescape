// Package config loads HomeScape configuration from environment variables and flags.
// Footprint-first: no external config-file format in v1 — env + flags only.
package config

import (
	"flag"
	"os"
	"strings"
	"time"
)

// Config holds the resolved runtime configuration.
type Config struct {
	Addr         string        // listen address, e.g. ":8080"
	DataDir      string        // directory for the SQLite database and other state
	DockerSocket string        // path to the Docker socket (used in F3 auto-discovery)
	LogLevel     string        // debug | info | warn | error
	SecretKey    string        // HS_SECRET_KEY passphrase; required only to store/read secrets
	PollInterval time.Duration // how often connectors are polled
}

// Load resolves configuration from flags first, falling back to environment
// variables, then to sensible container defaults.
func Load(args []string) Config {
	fs := flag.NewFlagSet("homescape", flag.ContinueOnError)

	addr := fs.String("addr", env("HS_ADDR", ":8080"), "listen address")
	dataDir := fs.String("data-dir", env("HS_DATA_DIR", "/data"), "data directory")
	dockerSocket := fs.String("docker-socket", env("HS_DOCKER_SOCKET", "/var/run/docker.sock"), "docker socket path")
	logLevel := fs.String("log-level", env("HS_LOG_LEVEL", "info"), "log level: debug|info|warn|error")
	pollInterval := fs.Duration("poll-interval", envDuration("HS_POLL_INTERVAL", 30*time.Second), "connector poll interval")

	// Ignore parse errors for unknown flags so the binary stays forgiving in containers.
	_ = fs.Parse(args)

	return Config{
		Addr:         *addr,
		DataDir:      *dataDir,
		DockerSocket: *dockerSocket,
		LogLevel:     strings.ToLower(*logLevel),
		SecretKey:    env("HS_SECRET_KEY", ""),
		PollInterval: *pollInterval,
	}
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

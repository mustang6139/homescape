// Command homescape is the HomeScape server: a single self-contained binary that serves
// the embedded frontend and the live API, backed by SQLite.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	homescape "github.com/MusiThang/homescape"
	"github.com/MusiThang/homescape/internal/config"
	"github.com/MusiThang/homescape/internal/connectors"
	"github.com/MusiThang/homescape/internal/discovery"
	"github.com/MusiThang/homescape/internal/scape"
	"github.com/MusiThang/homescape/internal/secret"
	"github.com/MusiThang/homescape/internal/server"
	"github.com/MusiThang/homescape/internal/store"
)

func main() {
	cfg := config.Load(os.Args[1:])
	log := newLogger(cfg.LogLevel)

	if err := run(cfg, log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return err
	}
	dbPath := filepath.Join(cfg.DataDir, "homescape.db")

	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer st.Close()

	// Seed the default Scape on a fresh install (L0 "just works").
	defaultSpec, err := scape.Marshal(scape.Default())
	if err != nil {
		return err
	}
	if _, err := st.Scapes().EnsureActive("Default", defaultSpec); err != nil {
		return err
	}

	// Secret vault: lazy — only required once a credential is actually stored.
	vault := secret.New(cfg.SecretKey, st.Secrets(), st.Settings())
	if vault.Enabled() {
		log.Info("secret encryption enabled")
	} else {
		log.Info("secret encryption disabled (HS_SECRET_KEY not set)")
	}

	// Graceful shutdown on SIGTERM/SIGINT — important inside containers.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Docker auto-discovery (opt-in, read-only). If the socket is unreachable, discovery is
	// simply disabled and the rest of the app runs normally.
	var disco *discovery.Discovery
	conns := []connectors.Connector{connectors.NewHTTPHealth()}
	dockerClient := discovery.Dial(cfg.DockerSocket)
	pingCtx, pingCancel := context.WithTimeout(ctx, 3*time.Second)
	if err := dockerClient.Ping(pingCtx); err != nil {
		log.Info("docker discovery disabled (socket unreachable)", "socket", cfg.DockerSocket)
	} else {
		disco = discovery.New(dockerClient, st.Integrations(), log)
		conns = append(conns, connectors.NewDockerStatus(disco))
		log.Info("docker discovery available", "socket", cfg.DockerSocket)
	}
	pingCancel()

	// Connector layer: registry of available connectors + a poller that fetches active
	// integrations and broadcasts resource updates over the shared SSE hub.
	registry := connectors.NewRegistry(conns...)
	hub := server.NewHub()
	src := &integrationSource{store: st, vault: vault, log: log}
	poller := connectors.NewPoller(registry, src, cfg.PollInterval, func(u connectors.ResourceUpdate) {
		hub.Broadcast(server.Event{Type: "resource.updated", Data: u})
	}, log)

	// A registry change (discovery) should trigger an immediate poll refresh.
	if disco != nil {
		disco.SetOnChange(func() { go poller.Refresh(ctx) })
	}

	srv := server.New(log, st, vault, hub, homescape.WebFS())

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go poller.Run(ctx)
	if disco != nil {
		go disco.Run(ctx)
	}

	go func() {
		log.Info("homescape listening", "addr", cfg.Addr, "data", cfg.DataDir)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}

func newLogger(level string) *slog.Logger {
	var lv slog.Level
	switch level {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lv}))
}

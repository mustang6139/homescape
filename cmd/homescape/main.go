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
	"github.com/MusiThang/homescape/internal/scape"
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

	srv := server.New(log, st, homescape.WebFS())

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown on SIGTERM/SIGINT — important inside containers.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

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

package main

import (
	"log/slog"

	"github.com/MusiThang/homescape/internal/connectors"
	"github.com/MusiThang/homescape/internal/secret"
	"github.com/MusiThang/homescape/internal/store"
)

// integrationSource adapts the registry + secret vault into the poller's Source: it lists
// active integrations and decrypts their secrets into ready-to-use connector Configs.
type integrationSource struct {
	store *store.Store
	vault *secret.Vault
	log   *slog.Logger
}

func (s *integrationSource) ActiveTargets() ([]connectors.Target, error) {
	all, err := s.store.Integrations().List()
	if err != nil {
		return nil, err
	}
	out := make([]connectors.Target, 0, len(all))
	for _, it := range all {
		if it.Status != "active" {
			continue
		}
		cfg := connectors.Config{
			BaseURL: it.BaseURL,
			// docker-status resolves live state from the discovery key.
			Options: map[string]any{"key": it.DiscoveryKey},
		}
		if it.SecretRef != "" {
			plain, err := s.vault.Open(it.SecretRef)
			if err != nil {
				// Can't decrypt (e.g. HS_SECRET_KEY missing/changed) — skip rather than fail
				// the whole cycle; the integration simply won't report until fixed.
				s.log.Warn("skipping integration: cannot decrypt secret", "integration", it.ID, "err", err)
				continue
			}
			cfg.Secret = string(plain)
		}
		out = append(out, connectors.Target{ID: it.ID, Type: it.Type, Config: cfg})
	}
	return out, nil
}

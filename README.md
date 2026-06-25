# HomeScape

A self-hosted homelab dashboard delivered as a single small container. Go backend + Svelte
frontend embedded in one static binary, backed by SQLite.

## Quick start

```bash
docker compose up --build      # → http://localhost:8080
# or
make dev                       # backend :8080 + Vite dev server (hot reload)
```

The dashboard works out of the box. Click **Customize** to change the theme/layout, add
service integrations, and enable Docker auto-discovery.

## Configuration

All configuration is via environment variables (or flags); there is no config file.

| Variable | Default | Purpose |
|---|---|---|
| `HS_ADDR` | `:8080` | Listen address |
| `HS_DATA_DIR` | `/data` | SQLite database + state directory (mount a volume) |
| `HS_DOCKER_SOCKET` | `/var/run/docker.sock` | Docker socket for auto-discovery |
| `HS_POLL_INTERVAL` | `30s` | How often connectors are polled |
| `HS_SECRET_KEY` | _(unset)_ | Passphrase to encrypt integration credentials at rest |

### Secrets

Integration API keys are encrypted at rest with AES-256-GCM; the key is derived from
`HS_SECRET_KEY` (scrypt + a per-install salt). It is **lazy**: only required once you add an
integration that stores a secret. Lose the passphrase → re-enter those credentials. Plain
HTTP health checks need no secret, so `HS_SECRET_KEY` is optional until you need it.

## Docker auto-discovery

Discovery is **opt-in** and **read-only**. Mount the socket read-only and enable it in
**Customize → Auto-discovery**:

```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock:ro
```

For tighter security, put a [docker-socket-proxy](https://github.com/Tecnativa/docker-socket-proxy)
in front and point `HS_DOCKER_SOCKET` at it. HomeScape only ever **reads** (list + events);
it never creates or execs containers.

### Container labels

| Label | Effect |
|---|---|
| `homescape.enable=true` | Opt the container in to discovery |
| `homescape.name` | Display name (default: container name) |
| `homescape.group` | Grouping |
| `homescape.url` | Health-check URL (enables HTTP latency/version checks) |
| `homescape.icon` | Icon hint |
| `homescape.id` | Explicit stable id (overrides the default key resolution) |

Without `homescape.url`, status comes from the container's running state (so a service is
"up" simply by running). Discovered services land in a review queue (or are added
automatically, depending on the mode).

## Architecture

- **Scape spec** — the dashboard state is a portable, declarative JSON document. Widgets
  reference integrations by stable handle, never by URL or secret, so a Scape stays
  shareable. Validated against `internal/scape/scape.schema.json`.
- **Services registry** — instance-local integrations (URL + encrypted secret + discovery
  state), separate from the spec.
- **Connectors** — produce normalized, typed resources (`service.status`, …) that widgets
  bind to. The poller fetches active integrations, caches the latest, and pushes changes
  over SSE; the frontend never calls services directly.

## Development

```bash
make build      # build frontend + embed into the Go binary
make test       # go test ./...  + frontend tests
make lint       # go vet + gofmt
make docker     # build the container image
```

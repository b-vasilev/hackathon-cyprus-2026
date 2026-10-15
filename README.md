# hackathon-cyprus-2026

ODYSSEY-2026-UNLEASH-THE-GODS

## Onboarding Factory foundation

One Go process and one local SQLite file. React is an empty scaffold.
No product features, GitHub/LLM integrations, auth or job workers are implemented yet.

```text
React / Vite (placeholder)
        |
        v
Go net/http -> SQLite + embedded migrations
```

### Setup and run

Install [mise](https://mise.jdx.dev/), then run from this repository:

```sh
mise trust
mise install
mise exec -- just setup
mise exec -- just backend::run
```

`mise install` installs pinned tools; `just setup` downloads backend dependencies. No secrets are needed.
The API defaults to `127.0.0.1:8080`; `.env.example` documents all settings.
Environment files are not loaded automatically. Override values in your shell, for example:

```sh
LOG_FORMAT=json HTTP_ADDR=127.0.0.1:9000 mise exec -- just backend::run
```

The database path is relative to `backend/` when using `just backend::run`.
Migrations run automatically at startup; errors stop startup.
Keep the SQLite file on a local persistent disk and use a single API instance.
Shutdown handles SIGINT/SIGTERM and drains HTTP requests before closing storage.

### API

| Route | Purpose |
| --- | --- |
| `GET /healthz` | Process liveness |
| `GET /readyz` | SQLite readiness; returns 503 on failure |
| `GET /api/openapi.json` | Embedded OpenAPI 3.1 contract |

All requests receive a server-generated `X-Request-ID`. CORS uses exact origins;
set `CORS_ORIGINS=` to disable cross-origin access. Requests from other origins return 403.
HTTP timeouts, a 1 MiB request body limit, panic recovery and structured logs are included.
Logs omit query strings and raw path parameters. Internal errors stay out of responses.
There is no authentication yet: only infrastructure routes are exposed. Add authentication and
product access checks before adding product data endpoints or exposing them publicly.
Use SSE for future job events. Streaming handlers will need a per-response write deadline.

### Checks

```sh
mise exec -- just check       # gofmt and inherited golangci-lint rules
mise exec -- just test        # short new foundation tests
mise exec -- just ci          # check, race tests, vulnerability audit, static build
```

Go 1.27.2, golangci-lint 2.14.0 and govulncheck 1.9.0 are pinned in `.mise.toml`.
The GitHub workflow runs the same backend checks and builds the Docker image.
Frontend is deliberately separate: see `frontend/README.md`.

```sh
docker compose up --build
```

Compose binds to localhost and keeps SQLite in the `onboarding-data` volume.
The image runs as a non-root user and contains no Node runtime or Antares binaries.

### Layout

```text
backend/cmd/api/              process entry point
backend/internal/app/        startup, migrations, graceful shutdown
backend/internal/api/        routes, middleware, OpenAPI
backend/internal/config/     validated environment settings
backend/internal/applog/     zerolog and request context
backend/internal/db/         SQLite connection and SQL migrations
frontend/                    empty React + Vite + TypeScript scaffold
```

Add product modules under `backend/internal/` and new SQL migration pairs under
`backend/internal/db/migrations/`. Next steps are product registration, source evidence,
a durable job queue, provider adapters and approved content revisions. Keep LLM requests
outside database transactions. Treat imported repository text as data, not instructions.

# Wayfinder

Generate a standard product passport from repository evidence. Each product gets
its own content, model, scenarios and diagrams through shared renderers.

```text
GitHub SHA + optional context
              |
      go-git + Repomix safety checks
              |
      Fantasy + Repomix MCP research
              |
  analyze + reconcile -> compose -> validate
              |
    versioned passport JSON + SQLite metadata
              |
     React Flow + ELK + evidence
```

The passport covers purpose, users, capabilities, domain, scenarios, architecture,
data, runtime, operations, development, decisions and ownership. Missing evidence
stays explicit. Generated revisions are drafts; they are not human approval.

## Run locally

Install [mise](https://mise.jdx.dev/), then:

```sh
mise trust
mise install
mise exec -- just setup
cp -n .env.example .env
# Generate once and save the value as CONNECTION_MASTER_KEY in .env.
openssl rand -hex 32
mise exec -- just dev
```

Both services run together; Ctrl+C stops both.

Open [the portal](http://127.0.0.1:5173). Add a connection, check its model,
register a product with GitHub URLs or context, then generate its passport.
`just` automatically loads the project-root `.env`; exported shell variables take precedence.
See `.env.example` for settings. Keep the master key outside version control and
back it up separately: losing it makes saved credentials unreadable.

Use a single API instance and a local persistent SQLite disk. Migrations run at
startup. One goqite worker processes durable queued runs. Interrupted running jobs
become failed after restart and can be generated again. A failed regeneration
keeps the last successful passport.

Passport content is stored in files, outside SQLite. `PASSPORT_CONTENT_DIR` defaults
to a `content` directory beside `DATABASE_PATH`. Local development keeps the database
and content in `backend/data/`, which is ignored by Git. Readable passport JSON lives under
`products/<product-id>/versions/<version>/`; collected source text and model responses
live under `runs/<run-id>/`. SQLite retains version metadata, paths, checksums,
connections, processing state and usage. Back up the database and content directory
together; the content directory must persist across API restarts and deployments.
Startup exports existing database payloads, verifies the files and clears the legacy
payload columns in one transaction. Failed exports preserve the database originals.
New files are immutable and published before their database references are committed.
Source settings and the frozen input configuration remain in SQLite.

System context diagrams use custom React Flow nodes and ELK-routed orthogonal
edges with labels included in layout. Narrow screens use a vertical arrangement;
other diagram kinds keep their existing renderer.

For a single-origin local build, run `just frontend::build`, then start the API
with `FRONTEND_DIR=../frontend/dist` and `CORS_ORIGINS=http://127.0.0.1:8080`.

## Accounts and access

Everyone signs in with an email and password. The first account registered on a
fresh database becomes an admin, so register it before sharing the portal URL.
Later accounts start with no product access.

| Role | Scope | Can |
| --- | --- | --- |
| Admin | Everything | Manage connections, products, all product access and admin roles. |
| Product lead | One product | Read and generate its passport; grant or revoke leads and learners. |
| Learner | One product | Read the product and its passport. |

A person can hold different roles on different products. Admins manage accounts and
every product's leads and learners under **People**. Leads manage their own products
from **People and access** on each product page. Sessions are HttpOnly cookies that last 30 days.

## LLM connections

Use any endpoint implementing a supported protocol:

| Protocol | SDK | Endpoint example |
| --- | --- | --- |
| OpenAI Responses | openai-go/v3 | `https://api.openai.com/v1` |
| OpenAI Chat Completions | openai-go/v3 | compatible gateway base URL |
| Anthropic Messages | anthropic-sdk-go | `https://api.anthropic.com` |

Model IDs are supplied by the operator. Each model selects native JSON Schema,
tool output, or JSON by instruction. All output is validated locally. Connections
support base or full URLs, a custom authentication header/prefix, additional
headers, query parameters, timeouts, concurrency and a shared quota group.
Native output requires support from the selected model and gateway.

Credentials are encrypted with AES-GCM and an Argon2id-derived key. APIs return
masked values; blank secret fields retain the saved value when editing. A job
freezes its connection revision, model and source configuration when queued.
Disabling a connection prevents new provider calls, including queued jobs.

Public HTTPS and local endpoints (localhost, loopback, and private networks) are
allowed by default. Other HTTP or special-network endpoints require an exact
trusted origin in `LLM_ALLOWED_PRIVATE_ORIGINS`. Redirects and environment proxies
are not followed. Destinations are DNS-checked and pinned for each connection.
Only allow destinations you control.

In the connection editor, use **Fetch models** to request the provider’s `/models`
list with the current URL, credentials, headers, and query parameters. You can
choose a returned ID or type one manually, then configure its output settings.

### Optional LLM sidecar

[`services/llm-proxy`](services/llm-proxy/README.md) is the existing separate
Codex OAuth proxy. Follow its setup instructions, then run:

```sh
docker compose up --build -d llm-proxy
```

In the portal, choose OpenAI Responses, the proxy's model ID and bearer key.
For a local API use `http://127.0.0.1:8317/v1` and allow origin
`http://127.0.0.1:8317`. For the Compose API use `http://llm-proxy:8317/v1`
and allow origin `http://llm-proxy:8317`. The profile stays off by default.

## Source collection and generation

GitHub sources resolve to immutable SHAs. All eligible repository text is indexed;
optional include patterns narrow that scope only when explicitly configured.
The embedded go-git library fetches the pinned commit without checking out or
running repository code. No installed Git executable is needed by the API or tests.
Regular source files are staged for the pinned Repomix CLI. Repomix
security and ignore filters determine safe files; original Git bytes are retained
for exact hashes and line citations.

In **Connections**, connect GitHub with a personal access token. The account is
shared across this portal and used for private source
repositories. Tokens are checked against GitHub, encrypted using `CONNECTION_MASTER_KEY`,
and never returned by the API. A fine-grained token needs Contents read access to
the selected repositories; organization approval may also be required.

Browser sign-in is implemented on the server; its UI button is temporarily hidden.
It requires a GitHub OAuth app. Set `GITHUB_OAUTH_CLIENT_ID`,
`GITHUB_OAUTH_CLIENT_SECRET`, and `GITHUB_OAUTH_REDIRECT_URL`. Register the exact
callback URL, ending in `/api/git-connection/oauth/callback`, in the OAuth app.
Compose and `just` read these settings from the project-root `.env`.
Use the portal's browser origin for the callback; HTTP is allowed only on localhost
or 127.0.0.1. OAuth requests the `repo` scope for private repository access and uses
state and PKCE. If an OAuth access token expires, reconnect in Connections.
Disconnect removes the saved credential; revoke provider access on GitHub if needed.
Optional `GITHUB_TOKEN` remains a server fallback when no account is connected.
Repomix stays installed and runs against the staged local files.

Safety ceilings: 5 sources, 20,000 files, 128 MiB total text and 8 MiB per file.
Oversized collections fail explicitly instead of silently using a partial catalog.
Dependencies, build outputs, binary files, secret-like paths, oversized files and
Repomix security exclusions are outside the indexed scope. The previous 80 KB / 40
document selection limit is no longer used by the application.
Source text can be sent to the selected provider; use appropriate source access.

Editable stage and section prompts live in
[`backend/internal/generation/prompts`](backend/internal/generation/prompts/README.md).
Each section has its own Markdown recipe. They are embedded at build time; rebuild
and restart to apply edits. The prompt version is recorded with every model call.

Fantasy runs the selected model through the saved gateway connection using OpenAI
Responses, OpenAI Chat Completions or Anthropic Messages. It can list files, search
the entire indexed source and batch paged reads through sandboxed Repomix MCP.
Repository code is never executed. Research stops after at most 24 model turns or
12 minutes and fails visibly if its completion/evidence checks are not satisfied.
Indexed, read and cited files are recorded separately, including original line
ranges. Every final evidence anchor must be inside a range actually read.

Research findings and original cited excerpts feed analysis with reconciliation,
composition and bounded repair. Analysis and reconciliation share one request.
The research recipe is editable in
[`backend/internal/research/prompts/research.md`](backend/internal/research/prompts/research.md).
The analysis recipe bounds the first model to its core responsibilities and flows;
the composition recipe keeps all 12 sections concise and reuses shared diagrams.
OpenAI connections can set a lower reasoning effort to reduce latency.
A bounded memory cache reuses safe source packages at unchanged commit SHAs;
restarting the API clears it. google/jsonschema-go checks each response.
Semantic checks verify all references, all 12 sections and citation line bounds.
React Flow renders model references; ELK computes coordinates. Generated content
cannot supply executable HTML, JavaScript or diagram code. Source material is
always treated as untrusted data, including instructions found inside documents.

## API and modules

The full [OpenAPI contract](backend/internal/api/openapi.json) is served at
`GET /api/openapi.json`. Request IDs, exact-origin CORS, timeouts, a 1 MiB body
limit, sanitized errors and structured logs come from the scaffold.

Generation progress is saved with ordered events in SQLite. The passport page
shows run and stage timing, recorded model usage, and an expandable activity feed.
`GET /api/generations/{id}/events` streams `snapshot` and `update` SSE events.
Each payload is a Run with only activity after the supplied cursor. Automatic
reconnect uses `Last-Event-ID`; a new connection can use `?after=<event-id>`.
Heartbeat comments keep idle streams open. The browser merges replayed events by
ID, falls back to five-second status polling while disconnected, and closes the
stream when the run finishes. A stream heartbeat does not mean worker progress.

| Routes | Purpose |
| --- | --- |
| `/healthz`, `/readyz` | Liveness and database readiness |
| `/api/factory/status` | Connection setup availability |
| `/api/llm-connections`, `/{id}`, `/{id}/test` | Save, edit, list and check connections |
| `/api/products`, `/{id}` | Register and inspect product sources |
| `/api/products/{id}/generations` | Queue generation; returns 202 |
| `/api/generations/{id}` | Stage, error, document count and token usage |
| `/api/products/{id}/passport`, `/passport-revisions` | Latest content and revision history |
| `/api/passport-revisions/{id}`, `/{id}/evidence/{evidenceId}` | Frozen content and exact source excerpts |

```text
backend/internal/source/          GitHub catalog, pinned Git + Repomix
backend/internal/llmgateway/      encrypted connections, network policy, limiter
backend/internal/llm/             official provider SDK adapters
backend/internal/passport/        shared contract and semantic validation
backend/internal/generation/      durable queue and generation stages
backend/internal/product/         product source configuration
backend/internal/api/             routes, OpenAPI and optional built UI
frontend/src/pages/               products, connections and passport
frontend/src/components/          shared diagrams, scenarios and evidence
```

The gateway adapts the vault, quota limiter and URL handling from Antares
`backend/internal/llmgateway`; its application-specific stores are not copied.

## Build and checks

```sh
mise exec -- just check
mise exec -- just test
mise exec -- just backend::test-race
mise exec -- just backend::audit
mise exec -- just check-web
mise exec -- just frontend::build
mise exec -- just generate-api  # after changing Go API contracts
```

OpenAPI types are generated with openapi-typescript, not maintained by hand.
CI configuration remains disabled; these commands run locally.

```sh
# Compose reads .env; set a stable CONNECTION_MASTER_KEY there first.
docker compose up --build api
```

Open [the built portal](http://127.0.0.1:8080). The image includes the Go API,
built UI, Node and Repomix, runs as a non-root user, and persists SQLite in the
`onboarding-data` volume. Source workspaces are private and removed after collection.

## MVP limits

This is a local, single-operator MVP. It has no login, product permissions or
publication approval workflow. Keep it on localhost; add authentication before
shared deployment. Source facts, inferred claims and unknowns are shown separately,
but an evidence link does not prove the model interpreted that evidence correctly.

Repomix 1.18.0 currently pulls a braces advisory through globby/micromatch. No
patched braces release was available during implementation. User glob patterns
are evaluated in Go; only escaped, catalog-verified exact paths reach Repomix.
Recheck `npm audit` when updating the packager dependency.

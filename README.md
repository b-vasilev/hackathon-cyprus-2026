# Onboarding Factory

Generate a standard product passport from repository evidence. Each product gets
its own content, model, scenarios and diagrams through shared renderers.

```text
GitHub SHA + optional context
              |
       LLM file selection
              |
      Git + Repomix safety checks
              |
  analyze -> reconcile -> compose -> validate
              |
    immutable passport in SQLite
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
# Generate once; keep this key stable across restarts.
export LLM_MASTER_KEY="$(openssl rand -hex 32)"
mise exec -- just backend::run
# In another terminal:
mise exec -- just frontend::dev
```

Open [the portal](http://127.0.0.1:5173). Add a connection, check its model,
register a product with GitHub URLs or context, then generate its passport.
Environment files are not loaded by `just`; export overrides in the shell.
See `.env.example` for settings. Keep the master key outside version control and
back it up separately: losing it makes saved credentials unreadable.

Use a single API instance and a local persistent SQLite disk. Migrations run at
startup. One goqite worker processes durable queued runs. Interrupted running jobs
become failed after restart and can be generated again. A failed regeneration
keeps the last successful passport.

For a single-origin local build, run `just frontend::build`, then start the API
with `FRONTEND_DIR=../frontend/dist` and `CORS_ORIGINS=http://127.0.0.1:8080`.

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

Public HTTPS is the default. HTTP or private endpoints require an exact trusted
origin in `LLM_ALLOWED_PRIVATE_ORIGINS`; redirects and environment proxies are
not followed. Public destinations are DNS-checked and pinned for each connection.
Only allow destinations you control.

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

GitHub sources resolve to immutable SHAs. A bounded file catalog is sent to the
selected model, which chooses exact files. Include patterns narrow that catalog.
Git fetches the pinned commit without checking out or running repository code.
Only selected regular files are staged for the pinned Repomix CLI. Repomix
security and ignore filters determine safe files; original Git bytes are retained
for exact hashes and line citations. Optional `GITHUB_TOKEN` supports private repos.

Limits: 5 sources, 120 documents, 300 KB total text, 128 KiB per file, 5,000 catalog
entries and 80 selected files per repository. Catalog truncation and source coverage
are explicit. These are context limits, not a claim of complete repository analysis.
Source text can be sent to the selected provider; use appropriate source access.

Editable stage and section prompts live in
[`backend/internal/generation/prompts`](backend/internal/generation/prompts/README.md).
Each section has its own Markdown recipe. They are embedded at build time; rebuild
and restart to apply edits. The prompt version is recorded with every model call.

The same selected model performs file selection, analysis, reconciliation,
composition and bounded repair. google/jsonschema-go checks each response.
Semantic checks verify all references, all 12 sections and citation line bounds.
React Flow renders model references; ELK computes coordinates. Generated content
cannot supply executable HTML, JavaScript or diagram code. Source material is
always treated as untrusted data, including instructions found inside documents.

## API and modules

The full [OpenAPI contract](backend/internal/api/openapi.json) is served at
`GET /api/openapi.json`. Request IDs, exact-origin CORS, timeouts, a 1 MiB body
limit, sanitized errors and structured logs come from the scaffold.

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
# Compose reads .env; set a stable LLM_MASTER_KEY there first.
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

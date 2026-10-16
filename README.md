# Wayfinder

Wayfinder is a product onboarding portal. Generate a product passport from
repository evidence, review and publish it for your team, and work through
Foundations lessons and quizzes. Each product gets its own content, model,
scenarios and diagrams through shared renderers.

```text
GitHub SHA + optional context
              |
      pinned source collection + safety checks
              |
      tool-driven source research
              |
   incremental model -> reconcile -> compose -> validate
              |
    versioned passport JSON + SQLite metadata
              |
     React Flow + ELK + evidence
```

The passport starts with product purpose and outcomes, repository relationships,
logical architecture, end-to-end flows and technologies. Useful supporting sections
cover data, runtime, development, operations, decisions and ownership. Missing evidence
stays explicit. Generated revisions start as drafts; a version is either draft or
published. In the product's Review tab, an admin or product lead can edit chapter
titles and text, comment on the passport or a chapter, flag a version for review
and publish or unpublish it. A flagged version cannot be published until the flag
is resolved. Edits to a published version stay unpublished until you publish the
changes; learners keep reading the published copy. Edits, comments, flags and
publication record the actor and time. Diagrams, flows, tables and evidence stay
as generated, and older section-based versions cannot be edited. Learners only
see the published version. Publishing stays inside the portal's existing product
access; it does not create a public link.
Product studio is available to admins and product leads. The separate Published
passports tab is available to all signed-in users for their assigned products.

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

Open [the portal](http://127.0.0.1:5173) and register the first account, which
becomes an admin. In **Connections**, add an LLM connection and check its model.
Connect GitHub there too if you need private repository access. In **Product
studio**, register a product with GitHub URLs or text context, then generate its
passport. Review and publish the draft before learners can read it, and grant
them product access under **People**.
`just` automatically loads the project-root `.env`; exported shell variables take precedence.
See `.env.example` for settings. Keep the master key outside version control and
back it up separately: losing it makes saved credentials unreadable.

Use a single API instance and a local persistent SQLite disk. Migrations run at
startup. A goqite worker pool processes durable queued runs in parallel.
`GENERATION_WORKERS` sets the number of workers (default `3`, positive integer).
Extra runs wait in the queue until a worker is available. Provider connection and
quota limits still apply to model requests. Interrupted running jobs
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

Context, container, component and repository diagrams use custom React Flow nodes
and ELK-routed orthogonal edges with labels included in layout. Narrow screens use
a vertical arrangement. Other diagram kinds use the standard React Flow renderer.
ELK layout runs in a web worker.

For a single-origin local build, run:

```sh
mise exec -- just frontend::build
PUBLIC_URL=http://127.0.0.1:8080 FRONTEND_DIR=../frontend/dist CORS_ORIGINS=http://127.0.0.1:8080 mise exec -- just backend::run
```

Open [the built portal](http://127.0.0.1:8080).

## Accounts and access

Everyone signs in with an email and password. The first account registered on a
fresh database becomes an admin, so register it before sharing the portal URL.
Later accounts start with no product access.

| Role | Scope | Can |
| --- | --- | --- |
| Admin | Everything | Manage connections, products, all product access and admin roles. |
| Product lead | One product | Read and generate its passport; grant or revoke leads and learners. |
| Learner | One product | Read the product and its published passport. |

A person can hold different roles on different products. Admins manage accounts and
every product's leads and learners under **People**. Leads manage their own products
from **People and access** on each product page. Sessions are HttpOnly cookies that last 30 days.

## Foundations

Every signed-in user can access Foundations, even without product assignments.
Modules contain lessons and quizzes; completing every lesson and passing the quiz
completes a module. Learning progress, quiz results and achievement unlocks are
saved to the account in SQLite and synced with a browser copy. The home page shows
the next learning step and available published passports.

## Connect AI

Use **MCP** (the gradient button in the top bar) to open Connect AI and copy
the portal's MCP URL into an AI app that supports remote MCP and OAuth browser
sign-in. Sign in to Wayfinder
when prompted and approve read access. Nothing needs downloading or copying from
the browser's session cookies. The connection follows your account's current
product permissions. The same page lists connected apps and lets you disconnect
each one; disconnecting immediately revokes access and token refresh.

The initial MCP server provides exactly two read-only tools:

| Tool | Result |
| --- | --- |
| `list_products()` | Accessible product IDs, names and published-passport availability. |
| `export_passports(product_ids?)` | Full published Markdown, filename, product ID, revision ID and version for each selected product. Omit IDs or pass `[]` for every accessible published passport. |

Exports are reading documents: chapter text, readable product descriptions and
referenced flows with their steps. Internal IDs, schema and snapshot metadata,
evidence, diagrams, raw model appendices, open questions and branding are omitted.
Filenames use the product name, such as `antares-v2.md` or `cli-anything-v1.md`;
colliding filenames in a batch receive a numeric suffix. Exports preserve
the published copy when reviewers have unpublished edits. There is no server-side
truncation, search, repository browsing, generation or publication through MCP.
Clients can save each returned `markdown` string using its `filename`.
The authenticated `GET /api/products/{id}/passport.md` route also downloads one
published passport directly using the portal login.

Set `PUBLIC_URL` to the browser-facing portal origin. Local development defaults
to `http://127.0.0.1:5173`; Vite proxies MCP, OAuth and discovery routes to the API.
For a single-origin build on port 8080, set `PUBLIC_URL=http://127.0.0.1:8080`.
Hosted deployments require an HTTPS origin and proxy the MCP/OAuth/discovery
routes to the API alongside `/api`. Include the portal origin in `CORS_ORIGINS`.

The server uses OAuth authorization codes with S256 PKCE, exact registered
callbacks, resource-bound access tokens and rotating refresh tokens. The only
scope is `passports:read`. Access tokens last one hour; app connections last up to
30 days before another sign-in is required. Only hashes of codes and tokens are
stored. Portal login sessions are not accepted as MCP credentials, and MCP tokens
cannot authenticate portal write routes. Remote clients must support public-client
dynamic registration; client metadata URL registration is not yet supported.

For a harness that launches local stdio servers, build the small adapter:

```sh
cd backend
go build -o bin/wayfinder-mcp ./cmd/mcp
```

Configure that binary as the harness's MCP command and optionally set
`WAYFINDER_MCP_URL` (default `http://127.0.0.1:5173/mcp`). The adapter opens browser
sign-in automatically and forwards only the two read-only tools. Its tokens stay
in memory, so restarting it asks for sign-in/consent again. Logs and sign-in links
go to stderr; stdout contains only MCP messages. Advanced clients may supply an
existing OAuth access token through `WAYFINDER_ACCESS_TOKEN` instead of browser
sign-in; the adapter does not refresh manually supplied tokens.

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
For a local API use `http://127.0.0.1:8317/v1`. For the Compose API use
`http://llm-proxy:8317/v1`. Both are allowed by the default local-network policy;
no extra `LLM_ALLOWED_PRIVATE_ORIGINS` entry is needed. The sidecar stays off
unless explicitly started.

## Source collection and generation

GitHub sources resolve to immutable SHAs. All eligible repository text is indexed;
optional include patterns narrow that scope only when explicitly configured.
The embedded go-git library fetches the pinned commit without checking out or
running repository code. No installed Git executable is needed by the API or tests.
Regular source files are staged for the pinned Repomix CLI. Repomix
security and ignore filters determine safe files; original Git bytes are retained
for exact hashes and line citations.

Admins can connect GitHub with a personal access token in **Connections**. The account is
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
Repomix security exclusions are outside the indexed scope.
Source text can be sent to the selected provider; use appropriate source access.

Editable stage and section prompts live in
[`backend/internal/generation/prompts`](backend/internal/generation/prompts/README.md).
Each section has its own Markdown recipe. They are embedded at build time; rebuild
and restart to apply edits. The prompt version is recorded with every model call.

Fantasy runs the selected model through the saved gateway connection using OpenAI
Responses, OpenAI Chat Completions or Anthropic Messages. It can list files, search
the entire indexed source and batch paged reads through sandboxed Repomix MCP.
Repository code is never executed. Research uses up to 20 workers with separate
source questions and a shared adaptive turn allowance. Model requests still obey
the saved connection's concurrency and quota limits. The allowance grows only when
new read-and-cited evidence reduces the number of open questions. Repeated reads,
classifications and renamed areas do not earn extensions. One recovery block lets
workers finish existing questions when progress stalls. Each research attempt has
a 60-minute limit, with the final 30 seconds reserved for saving and checking
results. The application sets no fixed turn cap. Research stops visibly with
unresolved questions if evidence checks cannot be satisfied.
Indexed, read and cited files are recorded separately, including original line
ranges. Every final evidence anchor must be inside a range actually read.
A deterministic inventory first identifies candidate source areas, manifests,
entry points, registries and cross-repository references. The researcher classifies
these candidates and records responsibilities, variants and handoffs. Evidence
obligations distinguish implementation, callers, contracts and validation, with
exact source ranges and separate gaps. Both endpoints of a cross-repository
handoff require support. The coverage check retains the unexplored frontier;
partial research is explicit. Inventory heuristics and read counts do not prove
that every subsystem was discovered or every claim is semantically correct.

The generation panel keeps research in one compact block: current area and question,
latest saved result, counts for the current plan, and expandable areas with answers
and source ranges. Details scroll inside the block. Budget decisions explain each
extension or stop. The plan and decisions survive reloads and live-stream reconnects;
counts describe the current plan, not a completion percentage of the whole product.

Research continues across early model stops while reachable source questions and
the attempt budget remain. Findings retain their original document IDs and read
ranges. A failed run can be retried through the generation endpoint by supplying
`resume_run_id` with its original connection and model. The retry verifies saved
source checksums instead of collecting repositories again. Accepted research is
rechecked and reused for downstream retries, retaining its complete or partial
status. Interrupted research continues from its saved findings and read ranges.
Each attempt saves a separate immutable findings artifact.

Analysis works through tools in small areas. It pages relevant findings, searches
the pinned source and reads original ranges, then saves objects, claims, relations,
scenarios and evidence into one canonical model. Each accepted edit and source read
is checkpointed before acknowledgement. Short context windows carry the current
question and paged indexes; earlier source bodies and the entire model are not
replayed in every request. Reconciliation resolves shared IDs and cross-area
contracts, then the server assembles the model. Progress reports the current area,
action and persisted entity counts. A failed analysis resumes its last valid
checkpoint. Per-run `request_timeout_seconds` can override the frozen connection's
execution timeout without changing its model or destination.

Composition uses this canonical model to author the document. It plans each
chapter, then generates small prose and diagram blocks with up to four concurrent
requests, within the connection's concurrency limit. Tables and scenario
references are assembled from the plans. Failed tasks retry independently while
preserving the validated model. The overview is written after the detailed
chapters and displayed first.
The research recipe is editable in
[`backend/internal/research/prompts/research.md`](backend/internal/research/prompts/research.md).
Analysis keeps the important responsibilities and variants in one canonical
model. Composition orders five core chapters: overview, repositories, logical
architecture, flows, then technologies. Useful supporting chapters cover data,
runtime, development and operations; decisions and ownership are conditional.
Users and capabilities normally belong in the overview, domain terms in data.

New revisions contain a version 2 document with freely ordered Markdown,
diagram references, scenario references and object-table blocks. The model writes
explanations between focused views; the portal supplies styling and interaction.
Objects, contracts, scopes and evidence remain shared across chapters. Markdown
is rendered without raw HTML or executable MDX. Older section-based revisions
continue to use their original renderer.
OpenAI connections can set a lower reasoning effort to reduce latency.
Source packages at unchanged commit SHAs are reused from memory and a disk cache
under `FACTORY_WORK_DIR/packages`. The disk cache survives API restarts. By default,
`FACTORY_WORK_DIR` is a `wayfinder` directory beside `DATABASE_PATH`.
google/jsonschema-go checks each response.
Semantic checks verify references, required core sections, content placement and
citation line bounds. Repository provenance is separate from runtime architecture.
React Flow renders model references; ELK computes coordinates. Generated content
cannot supply executable HTML, JavaScript or diagram code. Source material is
always treated as untrusted data, including instructions found inside documents.

### Product branding

Each generation looks for the analyzed product's brand guide, theme colours and
original logo in the collected sources. It follows up to three direct brand-guide
links and checks up to five linked logos, with a shared 30-second download budget.
Relative repository logo links use the analyzed commit and the existing GitHub
connection, including for private repositories. Only small, validated SVG, PNG,
JPEG and GIF images are accepted. Missing or unavailable branding uses Wayfinder
defaults; colours and logos are never invented.

An optional supplied brand kit takes priority over discovered colours. Discovered
logos can still fill a missing logo. Brand references and image bytes are frozen in
`runs/<run-id>/product-brand.json`, and the chosen theme is saved with the passport
revision and product card. The passport lists the sources used for its branding.
Brand accents apply to the card, passport heading, sections, diagrams and scenario
steps. Portal navigation, fonts, shared actions and semantic states remain shared.
Generate a new passport to discover branding for an existing product.

Admins and the product's leads can upload a custom product icon (SVG, PNG, JPEG
or GIF, up to 256 KB) from the product page. It replaces the discovered logo on
product tiles and the passport heading, and it survives regeneration. Uploaded
images pass the same safety checks as discovered logos.

## API and modules

The [OpenAPI contract](backend/internal/api/openapi.json) for the portal API is
served at `GET /api/openapi.json`. MCP and OAuth use separate endpoints. The API
includes request IDs, exact-origin CORS, timeouts, a 1 MiB body limit, sanitized
errors and structured logs.

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
| `/api/auth/register`, `/login`, `/logout`, `/me` | Account registration and cookie sessions; all under `/api/auth` |
| `/api/users`, `/{id}`, `/api/products/{id}/members`, `/members/{userId}` | Account administration and product access |
| `/api/me/learning` | Read and save the signed-in user's learning progress |
| `/api/factory/status` | Connection setup availability |
| `/api/llm-connections`, `/{id}`, `/{id}/test`, `/models` | Save, edit, list, check connections and fetch models |
| `/api/git-connection`, `/api/git-connection/oauth/start`, `/oauth/callback` | Shared GitHub credentials and GitHub browser sign-in; OAuth routes under `/api/git-connection` |
| `/api/products`, `/{id}` | Register and inspect product sources |
| `/api/products/{id}/icon` | Lead/admin upload or removal of a custom product icon |
| `/api/products/{id}/generations` | Queue generation; returns 202 |
| `/api/generations/{id}` | Stage, error, document count and token usage |
| `/api/generations/{id}/cancel`, `/{id}/events` | Cancel a run or stream its progress |
| `/api/products/{id}/passport`, `/passport-revisions` | Published content and revision history |
| `/api/passport-revisions/{id}`, `/{id}/evidence/{evidenceId}` | Frozen content and exact source excerpts |
| `/api/passport-revisions/{id}/publish`, `/{id}/unpublish` | Lead/admin publication of one version |
| `/api/passport-revisions/{id}/review`, `/{id}/comments`, `/{id}/flag`, `/{id}/chapters/{chapterId}` | Lead/admin review notes, flags and chapter edits |
| `/api/products/{id}/passport.md` | Download a published passport as Markdown |
| `/mcp`, `/oauth/*`, `/.well-known/*`, `/api/ai-connections` | Remote AI access, OAuth discovery, consent and app disconnection |

```text
backend/internal/source/          GitHub catalog, pinned Git + Repomix
backend/internal/llmgateway/      encrypted connections, network policy, limiter
backend/internal/llm/             official provider SDK adapters
backend/internal/passport/        shared contract and semantic validation
backend/internal/research/        source investigation, coverage and checkpoints
backend/internal/analysis/        tool-driven canonical model assembly
backend/internal/generation/      durable queue and generation stages
backend/internal/product/         product source configuration
backend/internal/access/          accounts, sessions and product roles
backend/internal/learning/        account learning progress
backend/internal/mcpauth/          OAuth clients, grants and tokens for MCP
backend/internal/api/             routes, OpenAPI and optional built UI
backend/cmd/mcp/                  local stdio adapter for remote MCP
frontend/src/foundations/         lessons, quizzes, progress and achievements
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
mise exec -- just frontend::test
mise exec -- just frontend::build
mise exec -- just proxy::check
mise exec -- just proxy::test -race -count=1
mise exec -- just proxy::audit
mise exec -- just generate-api  # after changing Go API contracts
```

Root `check`, `test`, `audit`, `build` and `ci` recipes cover the backend.
`check-web` runs frontend formatting, lint and type checks; frontend tests and
builds and proxy checks run separately as shown above.

OpenAPI types are generated with openapi-typescript, not maintained by hand.
CI runs backend, frontend and LLM proxy tests, formatting and lint checks on pull
requests and pushes to `main`. It also checks frontend types and builds the UI,
and audits Go dependencies. It does not deploy the app.

```sh
# Compose reads .env; set a stable CONNECTION_MASTER_KEY there first.
# Override the development PUBLIC_URL copied from .env.example.
PUBLIC_URL=http://127.0.0.1:8080 docker compose up --build api
```

Open [the built portal](http://127.0.0.1:8080). The image includes the Go API,
built UI, Node and Repomix, runs as a non-root user, and persists SQLite in the
`onboarding-data` volume. Passport content also persists in that volume, under
`/app/data/content`. Compose uses this default path; it does not forward
`PASSPORT_CONTENT_DIR` from `.env`. Temporary source workspaces are private and
removed after collection; cached source packages remain in `/app/data/wayfinder/packages`.

## Deployment

After `Tests and checks` succeeds for a push to `main`,
[the Deploy workflow](.github/workflows/deploy.yml) builds the tested commit's
image on GitHub, loads it on the Azure VM over SSH and restarts
[the production stack](deploy/compose.yaml): the API behind Caddy, which obtains and
renews the Let's Encrypt certificate. A manual run from the Actions tab bypasses
the test gate if needed.

| Item | Value |
| --- | --- |
| Portal | https://wayfinder-t1a.eastus2.cloudapp.azure.com |
| VM | `wayfinder-vm` (Ubuntu 24.04, Standard_B2s) in resource group `rg-bv-wayfinder`, East US 2 |
| Stack | `/opt/wayfinder` as `azureuser`; data in the `wayfinder_onboarding-data` volume |
| GitHub | `production` environment: secrets `DEPLOY_SSH_KEY`, `DEPLOY_KNOWN_HOSTS`; variables `DEPLOY_HOST`, `DEPLOY_USER` |

`/opt/wayfinder/.env` on the VM holds `SITE_HOST`, `CONNECTION_MASTER_KEY` and any
optional settings from `.env.example`; it is not in Git. Back up the master key
and the data volume. After editing `.env`, run `docker compose up -d` in `/opt/wayfinder`.
The production stack sets `PUBLIC_URL` and `CORS_ORIGINS` to `https://${SITE_HOST}`.
The MCP page shows `${PUBLIC_URL}/mcp`; OAuth discovery uses the same public origin.
Builds from the last week stay on the VM. To roll back, run
`docker tag wayfinder:<sha> wayfinder:current && docker compose up -d` there.

## Current limits

The portal supports multiple accounts, per-product roles, review and publication.
It runs as a single API instance with local SQLite and file storage; there is no
distributed worker setup. Drafts and unpublished edits are restricted to admins
and product leads. Publication gives access only to assigned users and admins;
it does not create a public passport URL. Remote AI access exports published
passports through the two read-only MCP tools.

Source facts, inferred claims and unknowns are shown separately, but an evidence
link does not prove the model interpreted that evidence correctly. Coverage and
unresolved questions remain explicit. Repomix is pinned in
`frontend/package.json`; check dependency advisories when updating it. User glob
patterns are evaluated in Go; only escaped, catalog-verified exact paths reach
Repomix.

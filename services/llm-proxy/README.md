# LLM sidecar

This separate service runs CLIProxyAPI's SDK for Codex OAuth subscriptions.
It has its own Go module and Docker image. The main app does not import or call it.
The SDK selects and rotates among the OAuth accounts in `auth-dir`.

## Docker

From the repository root, create a private credential folder and copy a Codex CLI
OAuth credential into it. The credential must use `auth_mode: "chatgpt"` and
contain access, ID, and refresh tokens. For example, after a Codex CLI login:

```sh
mkdir -p services/llm-proxy/auth
chmod 700 services/llm-proxy/auth
cp "${CODEX_HOME:-$HOME/.codex}/auth.json" services/llm-proxy/auth/auth.json
```

The folder is ignored by Git and Docker builds. It must be writable by container
UID/GID `1000:1000`; on Linux, set ownership of the dedicated folder and its files
with `sudo chown -R 1000:1000 services/llm-proxy/auth`.
The proxy writes refreshed tokens into this folder.

Create or update the root `.env` with:

```dotenv
LLM_SUBSCRIPTION_KEY=replace-with-a-long-random-private-key
LLM_SUBSCRIPTION_AUTH_HOST_DIR=./services/llm-proxy/auth
LLM_SUBSCRIPTION_PORT=8317
```

Generate a key with `openssl rand -hex 32`. An existing credential directory can
be used instead by setting `LLM_SUBSCRIPTION_AUTH_HOST_DIR` to its absolute path.
It must already exist. Keep credentials and the service key private.

```sh
docker compose up --build -d llm-proxy
docker compose ps llm-proxy
```

The service is optional: plain `docker compose up` starts only the app.
To start both, run `docker compose --profile llm-proxy up --build -d`.
Compose reads `.env` automatically. The service refuses to start without a key;
its Docker health check requires an authenticated `/v1/models` response with at
least one model.

Clients use `http://127.0.0.1:8317/v1` on the host, or
`http://llm-proxy:8317/v1` from another Compose container, with
`Authorization: Bearer <LLM_SUBSCRIPTION_KEY>`.
The host port is bound only to localhost. Management endpoints are disabled.

```sh
curl -H "Authorization: Bearer $LLM_SUBSCRIPTION_KEY" http://127.0.0.1:8317/v1/models
```

Export the key in your shell before using this curl example; `.env` is read by
Compose, not by curl. Stop the sidecar with `docker compose stop llm-proxy`.

## Local development

From the repository root:

```sh
mise exec -- just proxy::build
mise exec -- just proxy::test
mise exec -- just proxy::check
```

To run the local binary, set the key and credential folder explicitly:

```sh
LLM_SUBSCRIPTION_KEY=your-private-key \
LLM_SUBSCRIPTION_AUTH_DIR="$PWD/services/llm-proxy/auth" \
  services/llm-proxy/bin/llm-proxy -config services/llm-proxy/config.example.yaml
```

The example config listens on all interfaces for Docker. For host-only local
use, copy it to an ignored `services/llm-proxy/config.local.yaml`, set
`host: "127.0.0.1"`, and pass that file to `-config`.
Custom configs can supply `api-keys` and `auth-dir` when environment variables
are unset. The built-in `-healthcheck` uses `LLM_SUBSCRIPTION_KEY` and port 8317.

## Codex accounts

On startup and once per minute, the wrapper reads Codex CLI JSON files and creates
one SDK credential per account. When several CLI files have the same account ID,
the newest `last_refresh` seeds the proxy credential. Original CLI files keep
their format; existing proxy credentials under any name are reused.

The SDK refreshes proxy credentials. The wrapper writes refreshed tokens and
`last_refresh` back to every matching CLI file and imports newer CLI tokens into
its proxy credential. Other JSON fields are preserved. Equal timestamps with
different tokens produce a conflict instead of silently choosing a side.
Periodic reconciliation leaves undated files alone when a proxy credential
already exists. Avoid refreshing the same account in multiple processes at once.
Copies outside the mounted folder are not synchronized.

See [UPSTREAM.md](UPSTREAM.md) for the pinned dependency and license.

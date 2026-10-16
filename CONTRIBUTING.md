# Contributing

## Setup

Follow [Run locally](README.md#run-locally): install mise, run `mise install`, copy
`.env.example` to `.env`, set `CONNECTION_MASTER_KEY` and start both services with
`mise exec -- just dev`.

## Before opening a pull request

Run the same checks as CI:

```sh
mise exec -- just ci            # backend: format, lint, race tests, audit, build
mise exec -- just check-web     # frontend: format, lint, types
mise exec -- just frontend::test
mise exec -- just frontend::build
```

- After changing a Go API type or route, run `mise exec -- just generate-api` and
  commit `backend/internal/api/openapi.json` and `frontend/src/api.generated.ts`.
  CI fails when either is stale.
- Database migrations live in `backend/internal/db/migrations/`. Fetch the latest
  `main` before adding one and use the next free version number; never renumber a
  migration that is already on `main`.
- UI changes follow the [brandkit](.agents/skills/brandkit.md).
- Add tests for behaviour that protects users or catches a meaningful regression.

## Pull requests

Keep each pull request focused, describe what changed and how you verified it, and
wait for the `Tests and checks` workflow to pass. Merges to `main` deploy
automatically.

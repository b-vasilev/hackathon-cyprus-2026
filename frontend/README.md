# React placeholder

React + Vite + TypeScript only. The entry point renders an empty root; no screens or product logic exist.
ESLint, Biome and TypeScript rules come from Antares, with unused shared UI and test settings removed.
Run `mise exec -- just frontend::deps` once, then `mise exec -- just frontend::dev` from the repository root.
`mise exec -- just frontend::check` checks formatting, lint and types.
`mise exec -- just frontend::build` builds the placeholder.
Vite proxies `/api`, `/healthz` and `/readyz` to the local Go server.
The current Go Docker image only serves the API. Static asset embedding and generated API types are future work.

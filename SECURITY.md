# Security

## Reporting a vulnerability

Please report vulnerabilities privately to the repository maintainers rather than
in a public issue. Include the affected route or component, steps to reproduce and
the impact you observed. We aim to acknowledge reports within three working days.

## How the portal protects data

- **Accounts.** Passwords are hashed with bcrypt; sessions are random tokens stored
  only as SHA-256 hashes, sent in HttpOnly, SameSite=Lax cookies that are Secure
  behind HTTPS.
- **Access.** Every product route checks the caller's role. Learners see only
  published passports and practice; source settings, drafts and review data are
  limited to product leads and admins.
- **Credentials.** LLM API keys and GitHub tokens are personal to each account,
  encrypted with AES-GCM under a key derived from `CONNECTION_MASTER_KEY` with
  Argon2id and never returned by the API. Deleting an account removes its GitHub
  token and blanks its model connection secrets.
- **Outbound requests.** LLM destinations are DNS-checked and pinned; redirects and
  environment proxies are not followed. Loopback and private-network endpoints are
  allowed for self-hosted gateways, so run the portal where those networks hold
  nothing you would not expose to its users.
- **Source code.** Repositories are read at a pinned commit; their code is never
  executed. Portal users' glob patterns are evaluated in Go; Repomix receives only
  escaped, catalog-verified file paths.
- **AI access.** The MCP server uses OAuth with PKCE, exact redirect URIs, one-time
  codes, hashed and rotating tokens, and exports only published passports.
- **HTTP.** Exact-origin CORS, a 1 MiB request body limit, sanitized errors, a
  strict Content Security Policy for the built UI, and `X-Frame-Options: DENY`.

## Known limitations

- Sign-in has no rate limiting yet; put the portal behind a proxy that limits
  repeated login attempts when it is public.
- `npm audit` reports `braces` (GHSA-vfj7-8cjw-p6xm), pulled in through Repomix; no
  patched version exists. Portal users' glob patterns are evaluated in Go and never
  reach Repomix, but ignore files inside a collected repository (`.repomixignore`,
  `.ignore`) do. A crafted, deeply nested pattern there could exhaust the stack and
  fail that repository's collection; it does not expose data.
- Go dependencies are scanned with govulncheck in CI.

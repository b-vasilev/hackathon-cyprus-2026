# Working Rules

## Repository map

- `backend/` Go API: `internal/api` (routes, OpenAPI), `internal/generation`,
  `internal/research` and `internal/analysis` (passport pipeline), `internal/learningpack`
  (practice), `internal/access` (accounts, roles), `internal/db/migrations`.
- `frontend/` React + TypeScript portal: `src/pages`, `src/components`,
  `src/foundations` (lessons, quizzes, progress), `src/api.generated.ts` (generated).
- `deploy/` production compose and Caddy; `.github/workflows` CI and deploy.

## Commands

- Run locally: `mise exec -- just dev`.
- Checks: `just ci` (backend), `just check-web`, `just frontend::test`, `just frontend::build`.
- After changing API types or routes: `just generate-api`, then commit both generated files.

## Engineering Principles

- KISS: choose the simplest solution that satisfies the requirements.
- YAGNI: avoid speculative features, but implement required behavior for the general case across supported options and variants.
- DRY: avoid duplication, but don't abstract until duplication is meaningful.
- SOLID: use when it improves maintainability; don't over-engineer.
- Prefer composition over inheritance.
- Optimize for readability and maintainability.
- Keep modules loosely coupled and highly cohesive.

## Deliver a working result quickly
- Prioritize completing the task and getting it working.
- Keep changes focused. Avoid unrelated refactoring.
- Run only checks that help verify the actual change.
- Do not repeat successful checks unless relevant code changes.

## Keep tests minimal
- Add tests only when they protect important behavior or catch a meaningful regression.
- Do not write tests for every small change or trivial implementation detail.
- Prefer existing tests and focused checks over new test suites.
- Run the smallest relevant set of tests.

## Verify completion
- Before finishing, check every requirement in the user's request.
- Verify that the result works as intended using focused tests, direct inspection, or a manual check.
- Fix any issues found. Do not stop at a partial solution.
- Report what changed, what was verified, and any remaining limitations.
- Never claim success without evidence.

## Database migration ordering

- Before pushing a branch with database migrations or merging its PR, fetch the latest remote `main` and carefully check its highest migration version number. Do not rely on a stale local branch.
- Ensure new migrations have unique version numbers after the latest migration on remote `main`, in the correct dependency order. Resolve numbering conflicts and update affected references before pushing or merging.
- Recheck before merging even if numbering was checked before pushing. Do not renumber migrations already on `main`.

## Portal brandkit

- Before creating or changing portal UI, styles, components, layouts, visual assets, or product themes, read and follow [the brandkit](.agents/skills/brandkit.md).
- Treat it as the source of truth for the portal's visual identity. Keep product branding within the scope defined there. Change the agreed identity only when the user explicitly requests it.

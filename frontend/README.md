# Portal

React, TypeScript and Vite with shared product passport rendering.

```sh
npm ci
npm run dev
```

The dev server binds to `127.0.0.1:5173` and proxies API traffic to port 8080.
`npm run build` creates `dist/` for the Go server or container.

- React Query loads API data and polls active generation runs.
- React Flow + ELK render model references and trace scenario steps.
- API types come from `npm run generate:api` (openapi-typescript).
- `npm run format:check`, `npm run lint`, `npm run typecheck` verify the UI.
- The portal follows `.agents/skills/brandkit.md` in the repository root.

See the root README for connection setup, source limits and the full workflow.

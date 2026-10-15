# Passport prompt library

Edit Markdown here to change generation behavior. No Go string edits are needed.
The files are embedded in the API binary; rebuild and restart after changes.

| File | Responsibility |
| --- | --- |
| `system.md` | Trust boundaries, evidence rules and output policy |
| `select-files.md` | Select useful files from a verified catalog |
| `analyze.md` | Extract the shared product model from source evidence |
| `reconcile.md` | Resolve duplication, conflicts and broken references |
| `compose.md` | Turn the canonical model into standard passport views |
| `sections/*.md` | One recipe per section: goal, inputs, content, references, diagrams and unknowns |
| `repair-schema.md` | Correct a JSON Schema failure |
| `repair.md` | Correct semantic validation failures |
| `../../research/prompts/research.md` | Investigate the complete indexed repositories through search and Repomix MCP reads |

Section recipes are passed to analysis and composition, in the contract's fixed
order. Composition runs once for the whole passport, so sections share the same
model and IDs. A recipe is not a separate agent or a mandatory separate API call.

`prompts.go` loads the files; `service.go` supplies their data. The JSON contract
and semantic validators remain authoritative. Prompts cannot extend the schema.
Bump `PromptVersion` in `prompts.go` when changing behavior; each call records it.

Do not add product-specific facts or sample content to these files. All product
content must come from the selected sources through the configured model.

`research-architecture.md` applies the reference research method across all input
repositories: documentation checked against code, callers and contracts, meaningful
runtime boundaries, ordered workflows, agent acceptance gates, and explicit unknowns.
Repository boundaries never define architecture by themselves. The two Antares
repositories are a research reference, not hardcoded product content or mandatory inputs.

Communication references: [C4 diagrams](https://c4model.com/diagrams),
[C4 notation](https://c4model.com/diagrams/notation), and
[dynamic views](https://c4model.com/diagrams/dynamic). Static structure and execution
order are separate. The renderer owns visual conventions and graph layout.

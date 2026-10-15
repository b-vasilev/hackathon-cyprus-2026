# Passport prompt library

Edit Markdown here to change generation behavior. Files are embedded in the API
binary; the integrating chat owns rebuild/restart and end-to-end verification.

| File | Responsibility |
| --- | --- |
| information-architecture.md | Reading order, category questions and content boundaries |
| system.md | Trust boundary, exact evidence and safe structured output |
| select-files.md | Budgeted selection from a verified source catalog |
| analyze.md | One canonical evidence-backed model |
| incremental-analysis.md | Paged finding/source investigation, durable model parts and server finalization |
| reconcile.md | Identity, coverage, contradictions and cross-view consistency |
| research-architecture.md | Caller/implementation/contract checks and scoped structural views |
| compose.md | Ordered v2 document blocks over canonical facts |
| compose-plan.md | A short ordered plan for one chapter; no chapter prose |
| compose-part.md | Exactly one small prose or diagram block using the chapter recipe |
| sections/*.md | Goal, question, inputs, investigation, output, evidence/gaps, exclusions and acceptance |
| repair-schema.md | Schema repair without invented fields or evidence |
| repair.md | Legacy whole-passport repair reference, unused by chunked composition |
| repair-views.md | Legacy whole-Views repair reference, unused by chunked composition |
| ../../research/prompts/ | Source investigation owned separately from generation |

The passport is a concise architectural explanation over a detailed canonical
model. Composition uses soft starting sizes, groups related prose/table content,
and keeps supporting chapters short or absent. Architecture retains a whole-system
view and focused views of consequential complex subsystems; suggested sizes must
never flatten architectural levels or hide material boundaries. Complete repository
and technology coverage and canonical scenario accounting remain intact.

The recipe loader uses passport.ChapterIDs(): overview -> repositories ->
architecture -> scenarios -> technologies, then useful data/runtime/development/
operations and conditional decisions/ownership. Legacy users/capabilities normally
integrate into overview; domain integrates into data. Existing SectionIDs() and
saved v1 sections retain their compatibility contract. Recipes guide analysis and
composition together. Composition plans each chapter separately, then generates
its small prose and diagram blocks in a bounded worker pool. Both planning and
each block request receive that chapter's detailed recipe. Tables and scenario
references are assembled directly from the plan. Overview follows the detailed
chapters. A failed task retries independently; whole-document repair is not used.

New composition returns Views with sections=[] and document.version=2. Each
Chapter has id/title/question and freely ordered blocks. The only block kinds are
markdown, diagram_ref, scenario_ref and object_table. Object tables use objects,
repositories or technologies style. Markdown/table facts reference canonical
claims; diagram/scenario blocks reference shared definitions whose underlying
objects/relations/steps carry claims. Reference blocks have no claim_ids payload.
The exact JSON schema and semantic validators are authoritative.

Research first, reconcile detailed facts, synthesize overview last and read it
first. A full scenario has one home in scenarios, and a diagram is placed once.
Narrative explains scope/contracts and transitions between focused views.
Repository provenance, architectural hierarchy, static structure and temporal
behavior are different. A central complex engine needs actual important internals,
tools, contracts and statuses; per-view readability limits never cap the whole
model. Small single-repository products need no invented services or empty graphs.

Preserve independent inventory/coverage findings, exact revision-bound read
ranges and per-role gaps. An inventory, registration, manifest, test definition or
self-authored plan is not proof of complete behavior. Missing investigation is
not a product limitation. Generation uses supplied evidence and cannot claim it
performed missing research. Do not invent product-specific facts, commands,
owners, rationale, deployments or successful runs.

Document rendering accepts safe Markdown and known typed references, never
runtime-compiled MDX, HTML/JS/CSS, arbitrary embeds or generated layout code.
The application owns rendering, React Flow/ELK layout and product-card branding.
The current phase reuses Objects/Relations/Claims/Evidence and ordered Scenarios;
it does not introduce TechnologyUse, a full ADR/domain ontology or a flow engine.

The application installs analysis.Runner for tool-driven canonical analysis.
Its input contains the full pinned source collection and small finding records;
the model reads exact pages through tools rather than receiving a full context
replay. The integration retains research read ranges and merges analysis follow-up
reads into final coverage. Canonical checkpoints and their UI counts become visible
only after immutable content and a checksum-bound DB reference are committed.
After a provider failure the same run restores its last accepted checkpoint.
Document composition consumes the server-assembled model; individual part retries
cannot rewrite canonical facts. The old Model-schema path is retained
for unconfigured embedded fixtures; researched application runs require the runner.

prompts.go loads embedded files and records PromptVersion for reproducibility.
Bump it whenever behavior changes. Focused generation tests exercise loading and
the v2 pipeline; schema/semantic tests live with the passport contract. A passing
fixture verifies the contract, not the quality of an unseen live research run.

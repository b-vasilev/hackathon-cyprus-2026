# Reading sequence and chapter boundaries

Write one connected explanation for an engineer new to the product. Research and reconcile the detailed model first; synthesize overview last and place it first. Recipes guide investigation during analysis and selection during composition. Composition cannot perform missing research or create model facts.

Required v2 chapters, in this order:
1. overview / Product overview: purpose, actors, outcomes, input/output and product boundary. Optional small context view; no internal inventory or full scenario.
2. repositories / Repositories and connections: actual source owners, repository responsibilities and verified package/image/API/artifact contracts. Source structure is not runtime structure.
3. architecture / Logical architecture: whole-system static structure followed by important focused subsystem views. Explain real responsibilities, levels, boundaries and contracts; expand an opaque central engine.
4. scenarios / End-to-end flows: triggers, ordered stages, acceptance/terminal outcomes and a material variant matrix. Structure comes before temporal behavior.
5. technologies / Technologies and tools: full technology-to-responsibility reference after flows; small technology labels may already orient architecture readers.

Then include only useful evidenced supporting chapters, in this order:
6. data / Data and artifacts: vocabulary, invariants, artifact contracts, lineage, persistence and source of truth.
7. runtime / Runtime and deployment: execution forms, environment placement, startup/configuration and supported modes.
8. development / Development guide: source entry map, prerequisites, verified command definitions and focused checks for a first change.
9. operations / Operations and recovery: signals, failures, recovery and safety conditions.
10. decisions / Decisions and tradeoffs: recorded rationale, constraints, alternatives and consequences; conditional appendix.
11. ownership / Ownership and open questions: declared responsibilities, documented contact routes and substantive unresolved questions; conditional appendix.

Legacy optional categories users and capabilities normally belong in overview; domain belongs in data. Separate them only for substantial distinct content. Their stable titles are Users and access, Capabilities and limits, and Domain concepts. Do not create empty checklist chapters. Required chapters with missing evidence still state the specific gap in cited Markdown; v2 Chapter has no status field. Claim something is inapplicable only with positive evidence.

Every fact has one main home. Reuse shared identities without repeating inventories. Place each full scenario_ref only in scenarios and each diagram_ref in one chapter once; refer to its meaning in short prose elsewhere. Keep technology tables in technologies and source ownership tables in repositories. Data semantics, temporal behavior, deployment placement, setup and recovery have distinct questions. Use narrative before/after diagrams to explain scope and lead to the next reader question.

The v2 document orders Markdown, diagram_ref, scenario_ref and object_table blocks freely. Use this flexibility for useful explanations, not a fixed repeated card sequence. The canonical model and evidence remain shared; the document is an authored view, never a second fact store. Stable category titles orient navigation; product-specific subheadings explain actual content.

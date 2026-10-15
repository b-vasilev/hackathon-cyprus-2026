# scenarios
Goal: let a reader trace important product outcomes end to end.
Use: entry points, handlers, services, persistence paths, external integrations, tests and usage docs.
Prepare: a few representative scenarios, each with a trigger/input, ordered steps, participating object IDs, output and supported failure conditions. Include the main happy path and meaningful supported exceptions.
Select: canonical scenario IDs and claims needed to explain their steps. Do not rewrite scenarios as unrelated section text.
Visual: component or data views containing all objects used in each scenario, so step selection can highlight the actual path.
Missing evidence: leave unsupported asynchronous behavior, retries and errors unknown. Do not invent a queue or background worker.
Research: trace actual entry points and callers across execution boundaries. Capture shared versus specialized stages, optional behavior, validation, retries, checkpoints and unresolved work only when implementation supports them. Separate artifact generation from deployment.
Agent workflows: explicitly trace prepared context -> agent task -> candidate output -> deterministic checks -> accepted result or repair. Explain which decisions use an LLM and which use deterministic code. Include this only for products that contain agent work.
Presentation: the numbered steps define execution order; the component diagram defines structure. Reuse model IDs without pretending dependency direction is temporal order. Explain each step's important input/output contract in its summary.

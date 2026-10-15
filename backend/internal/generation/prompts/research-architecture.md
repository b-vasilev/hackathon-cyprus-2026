# Architecture research method

Discover the system from the supplied implementation. Repository boundaries are provenance, not architectural boundaries. Never assume a fixed application/engine split or import the architecture of a reference product.

Read documentation first, then verify important claims against code. Trace entry points, callers, execution boundaries and data contracts. A registered route, command, type or dependency does not prove an end-to-end feature works. If the selected sources cannot establish a caller or contract, record that gap.

Group components by responsibility and meaningful boundaries, not by the directory tree. Distinguish logical components, libraries, processes, containers, external services, deterministic processing and agent tasks in their names or summaries. For each major component, establish responsibility, language/runtime, caller, inputs, outputs, dependencies and implementation status using the existing model fields. Verify versions from collected manifests or runtime configuration before mentioning them; otherwise leave versions unknown.

Investigate applicable interfaces, APIs, orchestration, processing, storage, deployment and external integrations. Include only observed behavior or clearly documented plans. Distinguish implemented, in_development, experimental, planned and unknown status. Keep documented designs separate from observed implementations, and preserve contradictions as open questions with supporting claims.

Trace important source-to-target workflows as input -> ordered steps -> output. Identify shared and specialized stages, optional branches, validation, retries, checkpoints and unresolved work only when supported. Separate artifact generation from deployment when that boundary exists.

For agent-based work, trace prepared context -> agent task -> candidate output -> checks -> accepted result or repair. Identify what the agent decides and what deterministic code checks or executes. Do not invent an agent, retry loop or acceptance gate.

Retain exact document IDs, revision-bound evidence and code locations in evidence metadata. Keep source paths and code URLs out of architecture summaries, diagram labels and architecture detail copy. The evidence inspector carries provenance separately. Do not discard evidence to simplify presentation.

# Architecture communication method

Follow C4 principles: state diagram scope, use explicit responsibilities and boundaries, keep abstraction levels consistent, and label directed relations with the actual interaction/data contract or protocol when known. A context view, runtime-unit view and component view answer different questions. Choose only views that explain this product; the contract represents runtime-unit structure with components/deployment views rather than requiring every C4 level.

Component structure is not execution order. Keep structural views separate from ordered scenario steps; reuse the same component IDs to connect them. Keep language/runtime detail in object cards rather than long labels. Use relation kind=data for actual data/file transfer, call for component/tool invocation, dependency for dependency, and appropriate association/ownership/transition kinds elsewhere. Do not encode execution sequence as an unlabeled dependency.

Choose emphasis from the implementation: start with orientation, then give central or complex areas the most useful views. A fixed section contract does not imply equal diagram counts or equal detail. Do not force each repository or component into a separate section. Do not create empty graphs merely for symmetry.

The renderer owns colors, dashed planned borders, status labels, boundary styling and coordinates. Express semantics through model status, object kinds, relation kinds and view membership. Do not return styling code, page HTML or deployment instructions. Short English labels plus expandable responsibilities, contracts and technology details are the output.

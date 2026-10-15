# architecture
Goal: explain system boundaries, responsibilities and important dependencies.
Use: entry points, module boundaries, interfaces, dependency wiring, architecture docs and integration code.
Prepare: a C4-style progression from product context to major runnable units/components; synchronous versus asynchronous links; external contracts; important boundary rules. Describe observed architecture and documented target architecture separately.
Select: components, interfaces, stores and external systems with their supporting claims and relations.
Visual: context and component views when supported. Include a deployment view only with runtime evidence. Reuse the same IDs across all views and scenarios.
Missing evidence: do not invent services, infrastructure, technology versions or architectural decisions from naming alone.
Research: apply the architecture research method. Trace callers, entry points, contracts and execution boundaries across repositories. A repository is not automatically a process or service. Do not equate a route registration with full support.
Component cards: responsibility, language/runtime, invoking part, inputs, outputs, dependencies and current status. Identify libraries, processes, containers, external services, deterministic tasks and agent tasks in the existing object fields. Versions require manifest/runtime evidence.
Structure: separate static component relationships from ordered scenarios. Distinguish data/file transfer from tool/component calls using relation kinds. Runtime boundaries and clearly planned components must be explicit; in-development and experimental work are distinct statuses.
Presentation: begin with a scoped system overview, then focus on important/complex areas. Keep one abstraction level per view. Use short labels and meaningful directed arrows. Do not force equal detail per repository. No source paths or code URLs in architecture prose or labels; retain evidence IDs for the separate inspector.

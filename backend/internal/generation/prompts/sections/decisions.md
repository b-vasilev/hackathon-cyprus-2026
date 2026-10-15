# decisions
Goal: preserve recorded rationale and constraints in a conditional appendix.
Question: Which significant decisions were recorded, why, and with what consequences?
Inputs: supplied ADR/design/requirements/decision records and cross-references, canonical affected objects, implementation corroboration and exact evidence.
Investigation: separate rationale, implementation choice, requirements, measured outcomes and unresolved questions. Preserve context, chosen option, explicitly discussed alternatives/consequences, original decision status and supersession. Corroborate current implementation without silently resolving document/code conflict.
Output: concise cited Markdown, usually a decision/constraint table linked by shared identities. State quality goals with documented conditions/metrics; distinguish targets from measured outcomes. Use existing decision objects only when useful; preserve original ADR status in text, not a new object-status enum.
Evidence/gaps: code proves a choice, not why it was made. Do not reconstruct an ADR, alternatives, approval, debt priority or business intent. Say rationale was not found in reviewed sources when that is the actual gap.
Exclusions: no repeated stack, architecture or operational inventory; no generic scalability/flexibility prose. Omit the chapter when only absent rationale could be listed.
Acceptance: every reason has its own support; recorded status/history remains intact; constraints and requirements never become demonstrated guarantees; affected components and evidence resolve.

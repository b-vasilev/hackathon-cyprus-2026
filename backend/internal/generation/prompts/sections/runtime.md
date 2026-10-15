# runtime
Goal: explain execution forms and supported environment requirements.
Question: What runs where, how does it start, and which modes/configuration does it require?
Inputs: canonical runtime units/environments, entrypoints, build/container/service/deployment definitions, configuration parsers, persistent-state and shutdown evidence.
Investigation: trace each important unit through startup/configuration/resources/state/shutdown. Distinguish browser app, library, process, short-lived tool, infrastructure container and external service. Verify materially different modes and both sides of external connections where available.
Output: environment/mode summary -> supported placement view or table -> startup/configuration requirements -> persistent state and shutdown -> precise gaps. Deployment diagrams need evidenced environment, placement and unit identity; separate environment views unless explicitly comparing.
Evidence/gaps: classify statements as documented/configured/observed independently of feature status. Names/purposes of configuration are useful; secret values are not. Local definitions never prove production hosting, scaling, readiness or resilience.
Exclusions: no duplicate business flow, logical architecture or setup tutorial. Architectural parent_id is not an invented environment-instance ontology; express placement with existing objects/relations only when supported.
Acceptance: each important runnable part has supported placement/mode or a scoped gap; libraries do not become services; startup order does not imply readiness; ephemeral and persistent state remain distinct.

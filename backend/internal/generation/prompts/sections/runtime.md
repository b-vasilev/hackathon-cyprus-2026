# runtime
Goal: explain how the product starts, runs and connects to its environment.
Use: build files, startup wiring, container/service definitions, deployment docs and configuration parsers.
Prepare: processes and environments; startup dependencies; configuration names and purpose without secret values; request/background execution; deployment topology and lifecycle where evidenced.
Select: environment, component, external-system and store objects.
Visual: deployment and component views only for observed or explicitly documented runtime structure. Label planned topology clearly.
Missing evidence: do not infer production hosting, scale, regions or availability from a local development file.
Distinguish a library dependency from a separate process, container or external service. A C4 runtime unit is not necessarily a Docker container. Verify runtime and tool versions from selected manifests/configuration. Mark in-development, experimental and planned deployment behavior separately from implementation and from evidence confidence.

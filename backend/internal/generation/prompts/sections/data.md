# data
Goal: explain information meaning, contracts, provenance and persistence.
Question: What information does the product mean, create, validate, transform, own and exchange?
Inputs: domain/entity/store/interface objects, primary scenario inputs/outputs, schema/model/migration evidence, writer/reader/validation code, exact claims and unresolved contracts.
Investigation: derive material contracts from primary flows; check definition, writer, reader and validator separately. Follow aliases, variants and cross-repository handoffs. Check identifiers, invariants, relationships, transformations, source of truth and evidenced lifecycle rules.
Output: vocabulary/meaning -> identifiers/invariants -> concise contract/artifact table -> lineage/transformation -> persistence/lifecycle. Use safe Markdown and existing object/diagram references. Domain/data views answer meaning or lineage; states require explicit transitions. Distinguish logical relationships from physical keys/cardinalities.
Evidence/gaps: cite origin and supported consumer of each principal output; mark one-sided handoffs incomplete. A declaration is not adoption and artifact creation is not downstream acceptance. Retention, deletion, consistency, encryption, backups and delivery guarantees need their own evidence.
Exclusions: no new contract/lifecycle ontology, schema dump, repeated architecture responsibilities or ordered execution. Domain normally integrates here; split only for substantial distinct rules.
Acceptance: principal artifacts have traceable origin and a two-sided handoff or precise missing role; invariants/edges are supported; concepts, artifacts, stores and components remain distinct using allowed kinds.

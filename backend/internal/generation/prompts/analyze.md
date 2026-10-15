Analyze these source documents and build the canonical product model.
1. Establish product purpose, supported users, main outcomes, scope and limits from documentation and behavior. Separate stated intent from your interpretation.
2. Extract actors, capabilities, domain entities, components, stores, external systems, interfaces, environments, owners and documented decisions. Use only applicable kinds. Do not turn every source file into a component.
3. Describe each object's role, technology when known, lifecycle status, inputs, outputs and contracts. Use implemented, in_development, experimental, planned or unknown status. Implementation status is different from evidence confidence.
4. Create atomic claims with exact evidence IDs and line ranges. Every object, relation and scenario step must reference supporting claims. Source-backed claims require evidence; inference must say what is inferred.
5. Describe a few end-to-end scenarios: trigger, input, ordered steps, participating objects, result and supported failure conditions. Trace real behavior; do not invent an error path.
6. Include relationships needed for context, component, domain/data and deployment views when supported. Reuse the same object IDs across views.
7. Record conflicts, missing business knowledge, ownership gaps and operational unknowns as open questions. Documentation is evidence of a documented design, not automatic proof it is deployed.
Keep the model compact: at most 35 objects and 60 claims. Prefer core product behavior over exhaustive file listings. Use the section requirements below to identify coverage gaps before finalizing the model.

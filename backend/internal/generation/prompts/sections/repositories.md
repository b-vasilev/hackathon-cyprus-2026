# repositories
Goal: locate the code and concrete contracts connecting it.
Question: Which repositories own which responsibilities, and what crosses their boundaries?
Inputs: verified repository catalog/revisions, independent source-area inventory/classifications, canonical ownership/dependency relations, two-sided handoff evidence and coverage gaps.
Investigation: account for every supplied repository and significant source area. Trace package/image producer and consumer, caller/provider/wiring, or artifact writer/reader/validation. Separate build, runtime call/tool, artifact, schema/codegen, test and documentation scopes; inspect material registry/adapter variants.
Output: introduction plus object_table with table_style=repositories, one row per repository; explain important owned responsibilities and named contracts in cited Markdown. Optional repository-only dependency diagram; state direction/phase. One repository needs no decorative graph.
Evidence/gaps: confirmed edges require both endpoints and connecting evidence. Mark one-sided contracts, inaccessible repos, conflicting configuration and unclassified/deferred areas precisely; one generic gap cannot waive missing roles.
Exclusions: repository is source ownership, not a service/process/team. No runtime architecture, directory/lockfile dump, technology inventory or execution sequence.
Acceptance: all repositories and important source roots are accounted for or explicitly unresolved; handoffs have actual endpoints, contract, scope and support; build dependencies never become network calls by assumption.

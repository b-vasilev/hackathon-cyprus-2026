# data
Goal: explain what information the product owns and how it moves.
Use: schemas, migrations, queries, model types, import/export code, storage adapters and data docs.
Prepare: key entities and stores; ownership and source of truth; identifiers and relationships; ingestion/transformation/output paths; documented retention, deletion and consistency rules; supported sensitive-data boundaries without reproducing secrets or personal records.
Select: entities, stores, interfaces and source-backed data relationships.
Visual: a domain/data relationship view or directed data-flow graph. Describe logical relations honestly; do not present inferred relations as declared foreign keys.
Missing evidence: mark retention, classification, backup and consistency guarantees unknown unless supported.
Trace important source-to-target routes from entry point to accepted output. Identify shared transformations, route-specific steps, intermediate artifacts, validation gates, persistence and handoff boundaries. Distinguish files/artifacts from calls to tools or components. Artifact creation does not establish deployment or downstream acceptance.
Keep schema names, contract purpose and ownership concise. Store exact source paths and revision anchors in evidence metadata instead of architecture/data diagram labels.

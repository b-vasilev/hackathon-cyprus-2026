CREATE TABLE llm_connections (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    current_revision INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL
);
CREATE TABLE llm_connection_revisions (
    connection_id TEXT NOT NULL REFERENCES llm_connections(id),
    revision INTEGER NOT NULL,
    encrypted_config BLOB NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (connection_id, revision)
);
CREATE TABLE products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    sources_json TEXT NOT NULL CHECK (json_valid(sources_json)),
    created_at TEXT NOT NULL
);
CREATE TABLE generation_runs (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES products(id),
    connection_id TEXT NOT NULL REFERENCES llm_connections(id),
    connection_revision INTEGER NOT NULL,
    model TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued','running','succeeded','failed')),
    stage TEXT NOT NULL,
    input_json TEXT NOT NULL CHECK (json_valid(input_json)),
    error_code TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    finished_at TEXT,
    FOREIGN KEY (connection_id, connection_revision) REFERENCES llm_connection_revisions(connection_id, revision)
);
CREATE UNIQUE INDEX generation_runs_one_active ON generation_runs(product_id) WHERE status IN ('queued','running');
CREATE INDEX generation_runs_product ON generation_runs(product_id, created_at);
CREATE TABLE source_snapshots (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES generation_runs(id),
    locator TEXT NOT NULL,
    revision TEXT NOT NULL,
    coverage_json TEXT NOT NULL CHECK (json_valid(coverage_json))
);
CREATE TABLE source_documents (
    id TEXT PRIMARY KEY,
    snapshot_id TEXT NOT NULL REFERENCES source_snapshots(id),
    path TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    content TEXT NOT NULL,
    line_count INTEGER NOT NULL
);
CREATE INDEX source_documents_snapshot ON source_documents(snapshot_id);
CREATE TABLE llm_calls (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES generation_runs(id),
    stage TEXT NOT NULL,
    attempt INTEGER NOT NULL,
    prompt_version TEXT NOT NULL,
    request_id TEXT NOT NULL DEFAULT '',
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL,
    output_json TEXT NOT NULL DEFAULT '',
    error_code TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
CREATE TABLE passport_revisions (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES products(id),
    run_id TEXT NOT NULL UNIQUE REFERENCES generation_runs(id),
    version INTEGER NOT NULL,
    schema_version TEXT NOT NULL,
    content_json TEXT NOT NULL CHECK (json_valid(content_json)),
    created_at TEXT NOT NULL,
    UNIQUE (product_id, version)
);
CREATE TABLE goqite (
    id TEXT PRIMARY KEY DEFAULT ('m_' || lower(hex(randomblob(16)))),
    created TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ')),
    updated TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ')),
    queue TEXT NOT NULL,
    body BLOB NOT NULL,
    timeout TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ')),
    received INTEGER NOT NULL DEFAULT 0,
    priority INTEGER NOT NULL DEFAULT 0
) STRICT;
CREATE TRIGGER goqite_updated_timestamp AFTER UPDATE ON goqite BEGIN
    UPDATE goqite SET updated = strftime('%Y-%m-%dT%H:%M:%fZ') WHERE id = old.id;
END;
CREATE INDEX goqite_queue_priority_created_idx ON goqite(queue, priority DESC, created);

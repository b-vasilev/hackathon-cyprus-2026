CREATE TABLE analysis_checkpoints (
    run_id TEXT NOT NULL REFERENCES generation_runs(id) ON DELETE CASCADE,
    revision INTEGER NOT NULL CHECK (revision > 0),
    content_path TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (run_id, revision)
);

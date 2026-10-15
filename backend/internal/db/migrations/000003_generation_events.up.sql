ALTER TABLE generation_runs ADD COLUMN started_at TEXT NOT NULL DEFAULT '';
ALTER TABLE generation_runs ADD COLUMN stage_started_at TEXT NOT NULL DEFAULT '';

CREATE TABLE generation_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL REFERENCES generation_runs(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    stage TEXT NOT NULL,
    message TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX generation_events_run_id ON generation_events(run_id, id);

ALTER TABLE passport_revisions ADD COLUMN content_path TEXT NOT NULL DEFAULT '';
ALTER TABLE passport_revisions ADD COLUMN content_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE source_documents ADD COLUMN content_path TEXT NOT NULL DEFAULT '';
ALTER TABLE llm_calls ADD COLUMN output_path TEXT NOT NULL DEFAULT '';
ALTER TABLE llm_calls ADD COLUMN output_hash TEXT NOT NULL DEFAULT '';

-- Startup exports legacy payloads and clears their old columns only after the
-- corresponding files are written, synced and verified. API contracts are unchanged.

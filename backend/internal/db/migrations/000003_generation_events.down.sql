-- Restore external payloads into their legacy columns before downgrading a
-- database that has already exported content. Back up the database and files together.
ALTER TABLE llm_calls DROP COLUMN output_hash;
ALTER TABLE llm_calls DROP COLUMN output_path;
ALTER TABLE source_documents DROP COLUMN content_path;
ALTER TABLE passport_revisions DROP COLUMN content_hash;
ALTER TABLE passport_revisions DROP COLUMN content_path;
DROP TABLE generation_events;
ALTER TABLE generation_runs DROP COLUMN stage_started_at;
ALTER TABLE generation_runs DROP COLUMN started_at;

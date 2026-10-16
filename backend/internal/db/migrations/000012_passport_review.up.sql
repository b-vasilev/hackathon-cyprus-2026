-- Reviewers edit the working copy; learners keep reading the published copy.
ALTER TABLE passport_revisions ADD COLUMN published_content_path TEXT NOT NULL DEFAULT '';
ALTER TABLE passport_revisions ADD COLUMN published_content_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE passport_revisions ADD COLUMN edited_by TEXT NOT NULL DEFAULT '';
ALTER TABLE passport_revisions ADD COLUMN edited_at TEXT NOT NULL DEFAULT '';
ALTER TABLE passport_revisions ADD COLUMN flag_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE passport_revisions ADD COLUMN flagged_by TEXT NOT NULL DEFAULT '';
ALTER TABLE passport_revisions ADD COLUMN flagged_at TEXT NOT NULL DEFAULT '';
CREATE TABLE passport_review_activity (
    id TEXT PRIMARY KEY,
    revision_id TEXT NOT NULL REFERENCES passport_revisions(id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    kind TEXT NOT NULL CHECK (kind IN ('comment','edit','flag','resolve','publish','unpublish')),
    chapter_id TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
CREATE INDEX passport_review_activity_revision ON passport_review_activity(revision_id, created_at);

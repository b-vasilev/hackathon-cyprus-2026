ALTER TABLE passport_revisions ADD COLUMN approved_by TEXT NOT NULL DEFAULT '';
ALTER TABLE passport_revisions ADD COLUMN approved_at TEXT NOT NULL DEFAULT '';
ALTER TABLE passport_revisions ADD COLUMN published_by TEXT NOT NULL DEFAULT '';
ALTER TABLE passport_revisions ADD COLUMN published_at TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN published_revision_id TEXT REFERENCES passport_revisions(id) ON DELETE SET NULL;

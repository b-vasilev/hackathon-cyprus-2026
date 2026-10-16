DROP TABLE passport_review_activity;
ALTER TABLE passport_revisions DROP COLUMN flagged_at;
ALTER TABLE passport_revisions DROP COLUMN flagged_by;
ALTER TABLE passport_revisions DROP COLUMN flag_reason;
ALTER TABLE passport_revisions DROP COLUMN edited_at;
ALTER TABLE passport_revisions DROP COLUMN edited_by;
ALTER TABLE passport_revisions DROP COLUMN published_content_hash;
ALTER TABLE passport_revisions DROP COLUMN published_content_path;

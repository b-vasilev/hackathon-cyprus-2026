ALTER TABLE learning_packs ADD COLUMN source_content_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE learning_packs ADD COLUMN published_source_content_hash TEXT NOT NULL DEFAULT '';

-- Only unedited passports have an unambiguous source for existing practice.
-- Practice for edited passports must be generated again before publication.
UPDATE learning_packs
SET source_content_hash = (SELECT content_hash FROM passport_revisions WHERE id = revision_id)
WHERE content_json <> '' AND revision_id IN (SELECT id FROM passport_revisions WHERE edited_at = '');

UPDATE learning_packs
SET published_source_content_hash = (
    SELECT COALESCE(NULLIF(published_content_hash, ''), content_hash)
    FROM passport_revisions WHERE id = revision_id
)
WHERE published_content_json <> '' AND revision_id IN (SELECT id FROM passport_revisions WHERE edited_at = '');

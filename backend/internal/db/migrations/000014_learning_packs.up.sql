-- One generated learning pack per passport version: product quizzes and a first-task mission.
-- Reviewers work on content_json; learners read the copy captured at publication.
CREATE TABLE learning_packs (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    revision_id TEXT NOT NULL UNIQUE REFERENCES passport_revisions(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('generating','ready','failed')),
    content_json TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    published_content_json TEXT NOT NULL DEFAULT '',
    published_by TEXT NOT NULL DEFAULT '',
    published_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX learning_packs_published ON learning_packs(product_id, published_at);

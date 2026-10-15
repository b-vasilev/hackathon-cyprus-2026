ALTER TABLE generation_runs ADD COLUMN request_timeout_seconds INTEGER NOT NULL DEFAULT 0
    CHECK (request_timeout_seconds = 0 OR request_timeout_seconds BETWEEN 20 AND 900);

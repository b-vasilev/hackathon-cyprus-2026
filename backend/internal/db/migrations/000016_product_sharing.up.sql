ALTER TABLE products ADD COLUMN shared_with_all INTEGER NOT NULL DEFAULT 0 CHECK (shared_with_all IN (0, 1));

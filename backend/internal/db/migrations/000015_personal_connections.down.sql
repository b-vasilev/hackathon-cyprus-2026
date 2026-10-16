ALTER TABLE github_connection RENAME TO personal_github_connection;
CREATE TABLE github_connection (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    login TEXT NOT NULL,
    auth_method TEXT NOT NULL CHECK (auth_method IN ('token', 'oauth')),
    encrypted_token BLOB NOT NULL,
    connected_at TEXT NOT NULL
);
INSERT INTO github_connection SELECT 1,login,auth_method,encrypted_token,connected_at FROM personal_github_connection
WHERE owner_id=(SELECT id FROM users WHERE role='admin' ORDER BY created_at,id LIMIT 1);
DROP TABLE personal_github_connection;
ALTER TABLE generation_runs DROP COLUMN owner_id;
DROP INDEX llm_connections_owner;
ALTER TABLE llm_connections DROP COLUMN owner_id;

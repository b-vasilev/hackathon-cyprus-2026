ALTER TABLE llm_connections ADD COLUMN owner_id TEXT NOT NULL DEFAULT '';
UPDATE llm_connections SET owner_id=COALESCE((SELECT id FROM users WHERE role='admin' ORDER BY created_at,id LIMIT 1),'');
CREATE INDEX llm_connections_owner ON llm_connections(owner_id);

ALTER TABLE generation_runs ADD COLUMN owner_id TEXT NOT NULL DEFAULT '';
UPDATE generation_runs SET owner_id=COALESCE((SELECT owner_id FROM llm_connections WHERE id=generation_runs.connection_id),'');

ALTER TABLE github_connection RENAME TO shared_github_connection;
CREATE TABLE github_connection (
    owner_id TEXT PRIMARY KEY NOT NULL,
    login TEXT NOT NULL,
    auth_method TEXT NOT NULL CHECK (auth_method IN ('token', 'oauth')),
    encrypted_token BLOB NOT NULL,
    connected_at TEXT NOT NULL
);
INSERT INTO github_connection(owner_id,login,auth_method,encrypted_token,connected_at)
SELECT COALESCE((SELECT id FROM users WHERE role='admin' ORDER BY created_at,id LIMIT 1),''),login,auth_method,encrypted_token,connected_at FROM shared_github_connection;
DROP TABLE shared_github_connection;

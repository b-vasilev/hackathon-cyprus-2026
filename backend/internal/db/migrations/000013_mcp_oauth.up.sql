CREATE TABLE mcp_clients (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    redirect_uris_json TEXT NOT NULL CHECK (json_valid(redirect_uris_json)),
    created_at TEXT NOT NULL
);
CREATE TABLE mcp_codes (
    code_hash TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES mcp_clients(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    redirect_uri TEXT NOT NULL,
    challenge TEXT NOT NULL,
    resource TEXT NOT NULL,
    expires_at TEXT NOT NULL
);
CREATE TABLE mcp_grants (
    id TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES mcp_clients(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    revoked_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX mcp_grants_user ON mcp_grants(user_id);
CREATE TABLE mcp_tokens (
    grant_id TEXT PRIMARY KEY REFERENCES mcp_grants(id) ON DELETE CASCADE,
    access_hash TEXT NOT NULL UNIQUE,
    refresh_hash TEXT NOT NULL UNIQUE,
    resource TEXT NOT NULL,
    access_expires_at TEXT NOT NULL,
    refresh_expires_at TEXT NOT NULL
);

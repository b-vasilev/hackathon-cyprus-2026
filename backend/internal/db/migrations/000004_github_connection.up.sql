CREATE TABLE github_connection (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    login TEXT NOT NULL,
    auth_method TEXT NOT NULL CHECK (auth_method IN ('token', 'oauth')),
    encrypted_token BLOB NOT NULL,
    connected_at TEXT NOT NULL
);

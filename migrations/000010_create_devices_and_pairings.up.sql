CREATE TABLE pairing_sessions (
  token_hash BLOB PRIMARY KEY NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NOT NULL
);

CREATE INDEX pairing_sessions_expiry_idx
ON pairing_sessions (expires_at);

CREATE TABLE devices (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  token_hash  BLOB NOT NULL UNIQUE,
  created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  revoked_at  TIMESTAMP
);

CREATE TABLE sessions (
  token   TEXT PRIMARY KEY,
  data    BLOB NOT NULL,
  expiry  TEXT NOT NULL
);

CREATE INDEX sessions_expiry_idx ON sessions (expiry);

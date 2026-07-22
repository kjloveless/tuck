CREATE TABLE IF NOT EXISTS users (
  id            INTEGER   PRIMARY KEY,
  created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  name          TEXT      NOT NULL,
  email         TEXT      NOT NULL UNIQUE COLLATE NOCASE,
  password_hash BLOB      NOT NULL,
  activated     INTEGER   NOT NULL CHECK (activated IN (0, 1)),
  version       INTEGER   NOT NULL DEFAULT 1 CHECK (version >= 1)
);

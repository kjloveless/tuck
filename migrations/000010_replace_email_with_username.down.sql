CREATE TABLE users_old (
  id            INTEGER   PRIMARY KEY,
  created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  name          TEXT      NOT NULL,
  email         TEXT      NOT NULL UNIQUE COLLATE NOCASE,
  password_hash BLOB      NOT NULL,
  activated     INTEGER   NOT NULL CHECK (activated IN (0, 1)),
  version       INTEGER   NOT NULL DEFAULT 1 CHECK (version >= 1)
);

INSERT INTO users_old (id, created_at, name, email, password_hash, activated, version)
SELECT id, created_at, '', username, password_hash, activated, version
FROM users;

DROP TABLE users;
ALTER TABLE users_old RENAME TO users;

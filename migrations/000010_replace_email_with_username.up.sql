CREATE TABLE users_new (
  id            INTEGER   PRIMARY KEY,
  created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  username      TEXT      NOT NULL UNIQUE COLLATE NOCASE CHECK (length(username) <= 40),
  password_hash BLOB      NOT NULL,
  activated     INTEGER   NOT NULL CHECK (activated IN (0, 1)),
  version       INTEGER   NOT NULL DEFAULT 1 CHECK (version >= 1)
);

INSERT INTO users_new (id, created_at, username, password_hash, activated, version)
SELECT id, created_at, email, password_hash, activated, version
FROM users;

DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

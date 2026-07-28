CREATE TABLE IF NOT EXISTS permissions (
  id    INTEGER PRIMARY KEY,
  code  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS users_permissions (
  user_id       INTEGER NOT NULL REFERENCES user ON DELETE CASCADE,
  permission_id INTEGER NOT NULL REFERENCES permissions ON DELETE CASCADE,
  PRIMARY KEY(user_id, permission_id)
);

-- add the two permissions to the table
INSERT INTO permissions (code)
VALUES
  ('movies:read'),
  ('movies:write');


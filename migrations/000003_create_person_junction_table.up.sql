CREATE TABLE IF NOT EXISTS person (
  image_id      INTEGER NOT NULL,
  name          TEXT    NOT NULL,
  PRIMARY KEY (image_id, name),
  FOREIGN KEY (image_id)
    REFERENCES images(id)
    ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS person_name_idx
ON person (name);

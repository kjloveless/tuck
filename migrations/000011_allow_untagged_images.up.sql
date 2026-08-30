DROP TRIGGER IF EXISTS images_fts_insert;
DROP TRIGGER IF EXISTS images_fts_update;
DROP TABLE IF EXISTS images_fts;

CREATE TABLE images_new (
  id          INTEGER PRIMARY KEY,
  created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  location    TEXT,
  year        INTEGER NOT NULL CHECK (year >= 1888),
  people      TEXT
    CHECK (
      json_valid(people)
      AND json_type(people) = 'array'
      AND json_array_length(people) BETWEEN 0 AND 5
    ),
  version     INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
  path        TEXT
);

INSERT INTO images_new (id, created_at, location, year, people, version, path)
SELECT id, created_at, location, year, people, version, path FROM images;

CREATE TABLE person_new (
  image_id INTEGER NOT NULL,
  name     TEXT NOT NULL,
  PRIMARY KEY (image_id, name),
  FOREIGN KEY (image_id) REFERENCES images_new(id) ON DELETE CASCADE
);

INSERT INTO person_new (image_id, name)
SELECT image_id, name FROM person;

DROP TABLE person;
DROP TABLE images;
ALTER TABLE images_new RENAME TO images;
ALTER TABLE person_new RENAME TO person;

CREATE INDEX person_name_idx ON person (name);

CREATE VIRTUAL TABLE images_fts
USING fts5(location, content='images', content_rowid='id');

INSERT INTO images_fts(rowid, location)
SELECT id, location FROM images;

CREATE TRIGGER images_fts_insert
AFTER INSERT ON images
BEGIN
  INSERT INTO images_fts(rowid, location) VALUES (new.id, new.location);
END;

CREATE TRIGGER images_fts_update
AFTER UPDATE OF location ON images
BEGIN
  UPDATE images_fts SET location = new.location WHERE rowid = new.id;
END;

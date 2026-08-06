DROP TRIGGER IF EXISTS images_fts_insert;
DROP TRIGGER IF EXISTS images_fts_update;

CREATE TRIGGER images_fts_insert
AFTER INSERT ON images
BEGIN
  INSERT INTO images_fts(rowid, location)
  VALUES (new.id, new.location);
END;

CREATE TRIGGER images_fts_update
AFTER UPDATE OF location ON images
BEGIN
  UPDATE images_fts
  SET location = new.location
  WHERE rowid = new.id;
END;

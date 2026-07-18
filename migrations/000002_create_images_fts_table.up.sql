-- create fts5 table for location on the images table
CREATE VIRTUAL TABLE IF NOT EXISTS images_fts 
USING fts5(
  location,
  content="images",
  content_rowid="id"
);

-- create triggers to keep the fts5 table up to date
CREATE TRIGGER IF NOT EXISTS images_fts_insert 
AFTER INSERT ON images 
BEGIN
  INSERT INTO images_fts(id, location) 
  VALUES (new.id, new.location);
END;

CREATE TRIGGER IF NOT EXISTS images_fts_update 
AFTER UPDATE OF location ON images 
BEGIN
  UPDATE images_fts 
  SET location = new.location
  WHERE id = new.id;
END;

CREATE TABLE images (
  id          INTEGER PRIMARY KEY,
  created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  location    TEXT,
  year        INTEGER NOT NULL CHECK (year >= 1888),
  people      TEXT 
    CHECK (
      CASE
        WHEN json_valid(people)
        THEN json_type(people) = 'array'
          AND json_array_length(people) BETWEEN 1 AND 5
        ELSE 0
      END
    ),
  version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1)
);

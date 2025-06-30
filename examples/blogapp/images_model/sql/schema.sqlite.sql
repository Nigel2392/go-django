CREATE TABLE images (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  path TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  file_size INTEGER CHECK (file_size >= 0),
  file_hash TEXT NOT NULL
);

-- Indexes (SQLite requires explicit index creation)
CREATE INDEX images_image_created_at ON images (created_at);
CREATE INDEX images_image_file_hash ON images (file_hash);
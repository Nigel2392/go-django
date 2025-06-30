CREATE TABLE images (
  id int unsigned NOT NULL AUTO_INCREMENT,
  title varchar(255) NOT NULL,
  path varchar(255) NOT NULL,
  created_at datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  file_size int unsigned DEFAULT NULL,
  file_hash varchar(40) NOT NULL,

  PRIMARY KEY (id),
  KEY images_image_created_at (created_at),
  KEY images_image_file_hash (file_hash),
  CONSTRAINT wagtailimages_image_chk_5 CHECK ((file_size >= 0))
) COMMENT 'readonly:id,created_at';
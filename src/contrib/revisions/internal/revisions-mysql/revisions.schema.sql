CREATE TABLE IF NOT EXISTS Revision (
    id              BIGINT       AUTO_INCREMENT PRIMARY KEY,
    object_id       TEXT         NOT NULL,
    content_type    VARCHAR(400) NOT NULL,
    data            TEXT         NOT NULL,
    created_at      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP -- Read-only
);

ALTER TABLE Revision ADD INDEX (object_id(255));
ALTER TABLE Revision ADD INDEX (content_type(255));
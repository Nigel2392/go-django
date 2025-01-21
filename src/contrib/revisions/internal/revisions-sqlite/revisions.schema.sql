CREATE TABLE IF NOT EXISTS Revision (
    id              INTEGER   PRIMARY KEY,
    object_id       TEXT      NOT NULL,
    content_type    TEXT      NOT NULL,
    data            TEXT      NOT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP -- Read-only
);

CREATE INDEX IF NOT EXISTS Revision_object_id ON Revision(object_id);
CREATE INDEX IF NOT EXISTS Revision_content_type ON Revision(content_type);
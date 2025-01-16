CREATE TABLE IF NOT EXISTS PageNode (
    id              INTEGER   PRIMARY KEY AUTOINCREMENT, -- Read-only
    
    title           TEXT      NOT NULL,
    path            TEXT      UNIQUE NOT NULL,
    depth           INTEGER   NOT NULL,
    numchild        INTEGER   NOT NULL,

    -- URL path for this node
    -- This is a field based on the slugified title
    -- It is used to generate the URL route
    url_path        TEXT      NOT NULL,

    -- Slugified title
    slug            TEXT      NOT NULL,

    -- Status flags:
    -- 0x01: Published
    -- 0x02: Hidden
    -- 0x04: Deleted
    status_flags    BIGINT    NOT NULL,

    -- The page ID that this node represents
    page_id         BIGINT    NOT NULL,

    -- The unique content type name for this node
    content_type    TEXT      NOT NULL,

    -- Latest revision ID
    latest_revision_id BIGINT,

    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, -- Read-only
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP -- Read-only
);

CREATE INDEX IF NOT EXISTS PageNode_path ON PageNode(path);
CREATE INDEX IF NOT EXISTS PageNode_page_id ON PageNode(page_id);
CREATE INDEX IF NOT EXISTS PageNode_type_name ON PageNode(content_type);

CREATE TABLE IF NOT EXISTS Revision (
    id              BIGINT    AUTO_INCREMENT PRIMARY KEY,
    object_id       BIGINT    NOT NULL,
    content_type    TEXT      NOT NULL,
    data            TEXT      NOT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, -- Read-only
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP -- Read-only
);

CREATE INDEX IF NOT EXISTS Revision_object_id ON Revision(object_id);
CREATE INDEX IF NOT EXISTS Revision_content_type ON Revision(content_type);
CREATE UNIQUE INDEX IF NOT EXISTS Revision_object_id_content_type ON Revision(object_id, content_type);
--  
--  CREATE TRIGGER IF NOT EXISTS PageNode_decrement_numchild
--  AFTER DELETE ON PageNode
--  FOR EACH ROW
--  BEGIN
--      UPDATE PageNode
--      SET numchild = numchild - 1
--      WHERE path LIKE CONCAT(SUBSTR(OLD.path, 0, LENGTH(OLD.path) - 3), '%') AND depth = OLD.depth - 1;
--  END;
--  
--  CREATE TRIGGER IF NOT EXISTS PageNode_increment_numchild
--  AFTER INSERT ON PageNode
--  FOR EACH ROW
--  BEGIN
--      UPDATE PageNode
--      SET numchild = numchild + 1
--      WHERE path LIKE CONCAT(SUBSTR(NEW.path, 0, LENGTH(NEW.path) - 3), '%') AND depth = NEW.depth - 1;
--  END;
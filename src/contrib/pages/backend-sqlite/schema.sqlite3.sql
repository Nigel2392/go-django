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
    status_flags    INTEGER    NOT NULL,

    -- The page ID that this node represents
    page_id         INTEGER    NOT NULL,

    -- The unique content type name for this node
    content_type    TEXT      NOT NULL,

    -- Latest revision ID
    latest_revision_id INTEGER,

    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, -- Read-only
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP -- Read-only
);

CREATE INDEX IF NOT EXISTS PageNode_path ON PageNode(path);
CREATE INDEX IF NOT EXISTS PageNode_page_id ON PageNode(page_id);
CREATE INDEX IF NOT EXISTS PageNode_type_name ON PageNode(content_type);
CREATE UNIQUE INDEX IF NOT EXISTS PageNode_path_depth ON PageNode(slug, depth);

-- What a workaround to auto- update the updated_at field...
CREATE TRIGGER IF NOT EXISTS PageNode_updated_at
    AFTER UPDATE ON PageNode FOR EACH ROW
    WHEN OLD.id = NEW.id OR OLD.id IS NULL
BEGIN
    UPDATE PageNode SET updated_at=CURRENT_TIMESTAMP WHERE id=NEW.id;
END;

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
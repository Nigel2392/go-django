CREATE TABLE IF NOT EXISTS PageNode (
    id              INTEGER   PRIMARY KEY AUTOINCREMENT, -- Read-only
    
    title           TEXT      NOT NULL,
    path            TEXT      UNIQUE NOT NULL,
    depth           INTEGER   NOT NULL,
    numchild        INTEGER   NOT NULL,
    status_flags    INTEGER   NOT NULL,
    page_id         INTEGER   NOT NULL,
    content_type    TEXT      NOT NULL,

    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, -- Read-only
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP -- Read-only
);

CREATE INDEX IF NOT EXISTS PageNode_path ON PageNode(path);
CREATE INDEX IF NOT EXISTS PageNode_page_id ON PageNode(page_id);
CREATE INDEX IF NOT EXISTS PageNode_type_name ON PageNode(content_type);
CREATE TABLE IF NOT EXISTS PageNode (
    id              INTEGER   PRIMARY KEY AUTOINCREMENT,
    title           TEXT      NOT NULL,
    path            TEXT      UNIQUE NOT NULL,
    depth           INTEGER   NOT NULL,
    numchild        INTEGER   NOT NULL,
    status_flags    INTEGER   NOT NULL,
    page_id         INTEGER   NOT NULL,
    typeHash        TEXT      NOT NULL
);

CREATE INDEX IF NOT EXISTS PageNode_path ON PageNode(path);
CREATE INDEX IF NOT EXISTS PageNode_page_id ON PageNode(page_id);
CREATE INDEX IF NOT EXISTS PageNode_typeHash ON PageNode(typeHash);
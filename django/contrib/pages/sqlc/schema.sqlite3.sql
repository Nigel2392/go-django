CREATE TABLE PageNode (
    id        BIGINT  PRIMARY KEY AUTOINCREMENT,
    title     TEXT    NOT NULL,
    path      TEXT    UNIQUE NOT NULL,
    depth     BIGINT  NOT NULL,
    numchild  BIGINT  NOT NULL,
    page_id   BIGINT  NOT NULL,
    typeHash  TEXT    NOT NULL
);

CREATE INDEX PageNode_path ON PageNode(path);
CREATE INDEX PageNode_page_id ON PageNode(page_id);
CREATE INDEX PageNode_typeHash ON PageNode(typeHash);
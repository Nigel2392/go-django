CREATE TABLE PageNode (
    id BIGINT PRIMARY KEY AUTOINCREMENT,
    title TEXT,
    path  TEXT,
    depth BIGINT,
    numchild BIGINT,
    typeHash TEXT
);

CREATE INDEX PageNode_path ON PageNode(path);
CREATE INDEX PageNode_typeHash ON PageNode(typeHash);
CREATE TABLE PageNode (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    title TEXT,
    path  TEXT,
    depth BIGINT,
    numchild BIGINT,
    typeHash TEXT
);

CREATE INDEX PageNode_path ON PageNode(path);
CREATE INDEX PageNode_typeHash ON PageNode(typeHash);
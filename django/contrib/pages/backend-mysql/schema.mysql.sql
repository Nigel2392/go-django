CREATE TABLE IF NOT EXISTS PageNode (
    id              BIGINT       AUTO_INCREMENT PRIMARY KEY,
    title           VARCHAR(255) NOT NULL,
    path            VARCHAR(255) UNIQUE         NOT NULL,
    depth           BIGINT       NOT NULL,
    numchild        BIGINT       NOT NULL,

    -- URL path for this node
    -- This is a field based on the slugified title
    -- It is used to generate the URL route
    url_path        TEXT      NOT NULL,

    -- Status flags:
    -- 0x01: Published
    -- 0x02: Hidden
    -- 0x04: Deleted
    status_flags    BIGINT    NOT NULL,

    -- The page ID that this node represents
    page_id         BIGINT    NOT NULL,

    -- The unique content type name for this node
    content_type    VARCHAR(255) NOT NULL,

    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, -- Read-only
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP -- Read-only
);

ALTER TABLE PageNode ADD INDEX (path);
ALTER TABLE PageNode ADD INDEX (page_id);
ALTER TABLE PageNode ADD INDEX (content_type);
-- CREATE INDEX PageNode_path ON PageNode(path);
-- CREATE INDEX PageNode_page_id ON PageNode(page_id);
-- CREATE INDEX PageNode_type_name ON PageNode(content_type);
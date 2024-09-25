CREATE TABLE IF NOT EXISTS PageNode (
    id              BIGINT    AUTO_INCREMENT PRIMARY KEY,
    title           TEXT      NOT NULL,
    path            TEXT      UNIQUE         NOT NULL,
    depth           BIGINT    NOT NULL,
    numchild        BIGINT    NOT NULL,

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

    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, -- Read-only
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP -- Read-only
);

ALTER TABLE PageNode ADD INDEX (path);
ALTER TABLE PageNode ADD INDEX (page_id);
ALTER TABLE PageNode ADD INDEX (content_type);
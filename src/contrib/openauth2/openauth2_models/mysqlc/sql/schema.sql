CREATE TABLE IF NOT EXISTS oauth2_users (
    id                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    unique_identifier     VARCHAR(255) NOT NULL,
    data                  JSON NOT NULL,
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_administrator      BOOLEAN NOT NULL,
    is_active             BOOLEAN NOT NULL,
    PRIMARY KEY (id)
);
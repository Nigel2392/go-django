CREATE TABLE IF NOT EXISTS oauth2_users (
    id                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    unique_identifier     VARCHAR(255) NOT NULL,
    provider_name         VARCHAR(255) NOT NULL,
    data                  JSON NOT NULL,
    access_token          TEXT NOT NULL,
    refresh_token         TEXT NOT NULL,
    token_type            VARCHAR(60) NOT NULL,
    expires_at            DATETIME NOT NULL,
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_administrator      BOOLEAN NOT NULL,
    is_active             BOOLEAN NOT NULL,
    PRIMARY KEY (id)
);

ALTER TABLE oauth2_users ADD UNIQUE INDEX (unique_identifier(255));
ALTER TABLE oauth2_users ADD INDEX (provider_name(255));

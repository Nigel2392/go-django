CREATE TABLE IF NOT EXISTS oauth2_users (
    id                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    unique_identifier     VARCHAR(255) NOT NULL,
    provider_name         VARCHAR(255) NOT NULL,
    data                  JSON NOT NULL,
    access_token          TEXT NOT NULL,
    refresh_token         TEXT NOT NULL,
    expires_at            DATETIME NOT NULL,
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_administrator      BOOLEAN NOT NULL,
    is_active             BOOLEAN NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS oauth2_tokens (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id         BIGINT UNSIGNED NOT NULL,
    provider_name   VARCHAR(255) NOT NULL,
    data            JSON NOT NULL,
    access_token    TEXT NOT NULL,
    refresh_token   TEXT NOT NULL,
    expires_at      DATETIME NOT NULL,
    scope           TEXT,
    token_type      VARCHAR(255),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY (user_id, provider_name),
    FOREIGN KEY (user_id) REFERENCES oauth2_users(id) ON DELETE CASCADE
);
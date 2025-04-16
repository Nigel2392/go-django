CREATE TABLE IF NOT EXISTS oauth2_users (
    id                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT                                 COMMENT 'readonly:true',
    unique_identifier     VARCHAR(255) NOT NULL                                                   COMMENT 'readonly:true',
    provider_name         VARCHAR(255) NOT NULL                                                   COMMENT 'readonly:true',
    data                  JSON NOT NULL                                                           COMMENT 'readonly:true',
    access_token          TEXT NOT NULL                                                           COMMENT 'readonly:true',
    refresh_token         TEXT NOT NULL                                                           COMMENT 'readonly:true',
    token_type            VARCHAR(60) NOT NULL                                                    COMMENT 'readonly:true',
    expires_at            DATETIME NOT NULL                                                       COMMENT 'readonly:true',
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP                             COMMENT 'readonly:true',
    updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'readonly:true',
    is_administrator      BOOLEAN NOT NULL,
    is_active             BOOLEAN NOT NULL,
    PRIMARY KEY (id)
);

ALTER TABLE oauth2_users ADD UNIQUE INDEX (unique_identifier(255));
ALTER TABLE oauth2_users ADD INDEX (provider_name(255));

CREATE TABLE IF NOT EXISTS oauth2_users (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    unique_identifier TEXT NOT NULL,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_administrator  BOOLEAN NOT NULL,
    is_active         BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS oauth2_tokens (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL,
    provider_name   TEXT NOT NULL,
    data            TEXT NOT NULL,
    access_token    TEXT NOT NULL,
    refresh_token   TEXT NOT NULL,
    expires_at      DATETIME NOT NULL,
    scope           TEXT,
    token_type      TEXT,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, provider_name),
    FOREIGN KEY (user_id) REFERENCES oauth2_users(id) ON DELETE CASCADE
);

-- What a workaround to auto- update the updated_at field...
CREATE TRIGGER IF NOT EXISTS oauth2_users_updated_at
    AFTER UPDATE ON oauth2_users FOR EACH ROW
    WHEN OLD.id = NEW.id OR OLD.id IS NULL
BEGIN
    UPDATE oauth2_users SET updated_at=CURRENT_TIMESTAMP WHERE id=NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS oauth2_tokens_updated_at
    AFTER UPDATE ON oauth2_tokens FOR EACH ROW
    WHEN OLD.id = NEW.id OR OLD.id IS NULL
BEGIN
    UPDATE oauth2_tokens
    SET updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;
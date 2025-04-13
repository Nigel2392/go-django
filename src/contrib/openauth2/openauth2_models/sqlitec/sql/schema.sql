CREATE TABLE IF NOT EXISTS oauth2_users (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    unique_identifier TEXT NOT NULL,
    data              TEXT NOT NULL,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_administrator  BOOLEAN NOT NULL,
    is_active         BOOLEAN NOT NULL
);

-- What a workaround to auto- update the updated_at field...
CREATE TRIGGER IF NOT EXISTS oauth2_users_updated_at
    AFTER UPDATE ON oauth2_users FOR EACH ROW
    WHEN OLD.id = NEW.id OR OLD.id IS NULL
BEGIN
    UPDATE oauth2_users SET updated_at=CURRENT_TIMESTAMP WHERE id=NEW.id;
END;

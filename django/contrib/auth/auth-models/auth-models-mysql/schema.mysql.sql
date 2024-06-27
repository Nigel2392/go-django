CREATE TABLE IF NOT EXISTS `users` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `email` VARCHAR(255) NOT NULL,
    `username` VARCHAR(75) NOT NULL,
    `password` VARCHAR(256) NOT NULL,
    `first_name` VARCHAR(75) NOT NULL,
    `last_name` VARCHAR(75) NOT NULL,
    `is_administrator` BOOLEAN NOT NULL,
    `is_active` BOOLEAN NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `email` (`email`),
    UNIQUE KEY `username` (`username`)
);
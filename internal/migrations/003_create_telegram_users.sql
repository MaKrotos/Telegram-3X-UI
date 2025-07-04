-- +goose Up
CREATE TABLE IF NOT EXISTS telegram_users (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255),
    is_bot BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Индекс для быстрого поиска по telegram_id
CREATE INDEX IF NOT EXISTS idx_telegram_users_telegram_id ON telegram_users(telegram_id);

-- Индекс для поиска по username
CREATE INDEX IF NOT EXISTS idx_telegram_users_username ON telegram_users(username);

-- +goose Down
DROP INDEX IF EXISTS idx_telegram_users_username;
DROP INDEX IF EXISTS idx_telegram_users_telegram_id;
DROP TABLE IF EXISTS telegram_users; 
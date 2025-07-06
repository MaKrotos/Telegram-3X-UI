-- +goose Up
CREATE TABLE IF NOT EXISTS xui_servers (
    id SERIAL PRIMARY KEY,
    server_url VARCHAR(255) NOT NULL,
    server_name VARCHAR(255),
    server_location VARCHAR(255),
    server_ip VARCHAR(45),
    server_port INTEGER DEFAULT 54321,
    username VARCHAR(255),
    password VARCHAR(255),
    secret_key VARCHAR(255),
    two_factor_secret VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    added_by_tg_id BIGINT NOT NULL,
    added_by_username VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Индекс для быстрого поиска по URL сервера
CREATE INDEX IF NOT EXISTS idx_xui_servers_url ON xui_servers(server_url);

-- Индекс для поиска по telegram_id пользователя, который добавил
CREATE INDEX IF NOT EXISTS idx_xui_servers_added_by_tg_id ON xui_servers(added_by_tg_id);

-- Индекс для поиска по IP сервера
CREATE INDEX IF NOT EXISTS idx_xui_servers_ip ON xui_servers(server_ip);

-- Индекс для поиска по времени создания
CREATE INDEX IF NOT EXISTS idx_xui_servers_created_at ON xui_servers(created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_xui_servers_created_at;
DROP INDEX IF EXISTS idx_xui_servers_ip;
DROP INDEX IF EXISTS idx_xui_servers_added_by_tg_id;
DROP INDEX IF EXISTS idx_xui_servers_url;
DROP TABLE IF EXISTS xui_servers; 
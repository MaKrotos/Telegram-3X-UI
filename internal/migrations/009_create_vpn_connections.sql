-- +goose Up

CREATE TABLE IF NOT EXISTS vpn_connections (
    id SERIAL PRIMARY KEY,
    telegram_user_id BIGINT NOT NULL,
    username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    server_id INTEGER NOT NULL,
    inbound_id INTEGER NOT NULL,
    client_id VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL,
    vpn_login VARCHAR(255) NOT NULL,
    vpn_password VARCHAR(255) NOT NULL,
    vless_link TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (telegram_user_id) REFERENCES telegram_users(telegram_id) ON DELETE CASCADE,
    UNIQUE(telegram_user_id, inbound_id)
);

-- Комментарии к таблице
COMMENT ON TABLE vpn_connections IS 'VPN подключения пользователей Telegram';
COMMENT ON COLUMN vpn_connections.telegram_user_id IS 'Telegram ID пользователя';
COMMENT ON COLUMN vpn_connections.server_id IS 'ID XUI сервера';
COMMENT ON COLUMN vpn_connections.inbound_id IS 'ID inbound в XUI';
COMMENT ON COLUMN vpn_connections.client_id IS 'UUID клиента в XUI';
COMMENT ON COLUMN vpn_connections.email IS 'Email клиента в XUI';
COMMENT ON COLUMN vpn_connections.port IS 'Порт VPN подключения';
COMMENT ON COLUMN vpn_connections.vpn_login IS 'Логин для VPN';
COMMENT ON COLUMN vpn_connections.vpn_password IS 'Пароль для VPN';
COMMENT ON COLUMN vpn_connections.vless_link IS 'VLESS ссылка для подключения';
COMMENT ON COLUMN vpn_connections.is_active IS 'Активно ли подключение';

CREATE INDEX IF NOT EXISTS idx_vpn_connections_user_id ON vpn_connections(telegram_user_id);
CREATE INDEX IF NOT EXISTS idx_vpn_connections_server_id ON vpn_connections(server_id);
CREATE INDEX IF NOT EXISTS idx_vpn_connections_active ON vpn_connections(is_active);

-- +goose Down

DROP INDEX IF EXISTS idx_vpn_connections_active;
DROP INDEX IF EXISTS idx_vpn_connections_server_id;
DROP INDEX IF EXISTS idx_vpn_connections_user_id;
DROP TABLE IF EXISTS vpn_connections; 
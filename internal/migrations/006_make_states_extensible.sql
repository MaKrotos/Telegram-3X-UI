-- +goose Up

-- Создаем таблицу для определения состояний
CREATE TABLE IF NOT EXISTS user_states (
    id SERIAL PRIMARY KEY,
    state_code VARCHAR(50) UNIQUE NOT NULL,
    state_name VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    can_perform_actions BOOLEAN DEFAULT FALSE,
    can_manage_servers BOOLEAN DEFAULT FALSE,
    can_create_connections BOOLEAN DEFAULT FALSE,
    can_view_only BOOLEAN DEFAULT FALSE,
    requires_admin_approval BOOLEAN DEFAULT FALSE,
    auto_expire BOOLEAN DEFAULT FALSE,
    default_expiry_duration INTERVAL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создаем таблицу для определения ожидаемых действий
CREATE TABLE IF NOT EXISTS expected_actions (
    id SERIAL PRIMARY KEY,
    action_code VARCHAR(50) UNIQUE NOT NULL,
    action_name VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    priority INTEGER DEFAULT 0,
    auto_resolve BOOLEAN DEFAULT FALSE,
    auto_resolve_after INTERVAL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создаем таблицу для связи состояний и действий
CREATE TABLE IF NOT EXISTS state_action_mappings (
    id SERIAL PRIMARY KEY,
    state_code VARCHAR(50) NOT NULL REFERENCES user_states(state_code) ON DELETE CASCADE,
    action_code VARCHAR(50) NOT NULL REFERENCES expected_actions(action_code) ON DELETE CASCADE,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(state_code, action_code)
);

-- Создаем таблицу для истории изменений состояний
CREATE TABLE IF NOT EXISTS user_state_history (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES telegram_users(id) ON DELETE CASCADE,
    telegram_id BIGINT NOT NULL,
    old_state VARCHAR(50),
    new_state VARCHAR(50) NOT NULL,
    old_action VARCHAR(50),
    new_action VARCHAR(50),
    reason TEXT,
    changed_by_tg_id BIGINT,
    changed_by_username VARCHAR(255),
    expires_at TIMESTAMP,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Вставляем базовые состояния
INSERT INTO user_states (state_code, state_name, description, can_perform_actions, can_manage_servers, can_create_connections, can_view_only, requires_admin_approval, auto_expire, default_expiry_duration) VALUES
('active', 'Активный', 'Полный доступ к системе', TRUE, TRUE, TRUE, FALSE, FALSE, FALSE, NULL),
('inactive', 'Неактивный', 'Ограниченный доступ', FALSE, FALSE, FALSE, TRUE, FALSE, FALSE, NULL),
('blocked', 'Заблокированный', 'Полный запрет доступа', FALSE, FALSE, FALSE, FALSE, TRUE, FALSE, NULL),
('pending_verification', 'Ожидает верификации', 'Ограниченный доступ до верификации', FALSE, FALSE, FALSE, TRUE, FALSE, TRUE, INTERVAL '7 days'),
('suspended', 'Временно приостановлен', 'Временный запрет доступа', FALSE, FALSE, FALSE, FALSE, FALSE, TRUE, INTERVAL '24 hours'),
('deleted', 'Удаленный', 'Полный запрет доступа', FALSE, FALSE, FALSE, FALSE, TRUE, FALSE, NULL),
('premium', 'Премиум', 'Расширенные возможности', TRUE, TRUE, TRUE, FALSE, FALSE, FALSE, NULL),
('trial', 'Пробный период', 'Ограниченный доступ на время пробного периода', TRUE, TRUE, TRUE, FALSE, FALSE, TRUE, INTERVAL '30 days'),
('quarantine', 'Карантин', 'Временное ограничение для проверки', FALSE, FALSE, FALSE, TRUE, TRUE, TRUE, INTERVAL '48 hours'),
('maintenance', 'Техническое обслуживание', 'Временное ограничение доступа', FALSE, FALSE, FALSE, TRUE, FALSE, TRUE, INTERVAL '2 hours');

-- Вставляем базовые ожидаемые действия
INSERT INTO expected_actions (action_code, action_name, description, priority, auto_resolve, auto_resolve_after) VALUES
('none', 'Ничего не ожидается', 'Пользователь может использовать систему нормально', 0, FALSE, NULL),
('verify_email', 'Верификация email', 'Пользователь должен подтвердить email', 1, FALSE, NULL),
('complete_profile', 'Заполнение профиля', 'Пользователь должен дополнить информацию профиля', 2, FALSE, NULL),
('add_payment', 'Добавление платежа', 'Пользователь должен добавить способ оплаты', 3, FALSE, NULL),
('contact_support', 'Обращение в поддержку', 'Пользователь должен связаться с поддержкой', 4, FALSE, NULL),
('wait_approval', 'Ожидание одобрения', 'Пользователь ждет одобрения заявки', 5, FALSE, NULL),
('verify_phone', 'Верификация телефона', 'Пользователь должен подтвердить номер телефона', 1, FALSE, NULL),
('complete_kyc', 'Прохождение KYC', 'Пользователь должен пройти проверку личности', 3, FALSE, NULL),
('update_documents', 'Обновление документов', 'Пользователь должен обновить документы', 2, FALSE, NULL),
('resolve_dispute', 'Разрешение спора', 'Пользователь должен разрешить спор', 4, FALSE, NULL);

-- Создаем связи состояний и действий
INSERT INTO state_action_mappings (state_code, action_code, is_default) VALUES
('active', 'none', TRUE),
('inactive', 'complete_profile', TRUE),
('blocked', 'contact_support', TRUE),
('pending_verification', 'verify_email', TRUE),
('suspended', 'contact_support', TRUE),
('deleted', 'none', TRUE),
('premium', 'none', TRUE),
('trial', 'add_payment', TRUE),
('quarantine', 'wait_approval', TRUE),
('maintenance', 'none', TRUE);

-- Создаем индексы для новых таблиц
CREATE INDEX IF NOT EXISTS idx_user_states_code ON user_states(state_code);
CREATE INDEX IF NOT EXISTS idx_user_states_active ON user_states(is_active);
CREATE INDEX IF NOT EXISTS idx_expected_actions_code ON expected_actions(action_code);
CREATE INDEX IF NOT EXISTS idx_expected_actions_active ON expected_actions(is_active);
CREATE INDEX IF NOT EXISTS idx_state_action_mappings_state ON state_action_mappings(state_code);
CREATE INDEX IF NOT EXISTS idx_state_action_mappings_action ON state_action_mappings(action_code);
CREATE INDEX IF NOT EXISTS idx_user_state_history_user ON user_state_history(user_id);
CREATE INDEX IF NOT EXISTS idx_user_state_history_telegram ON user_state_history(telegram_id);
CREATE INDEX IF NOT EXISTS idx_user_state_history_created ON user_state_history(created_at);

-- +goose Down

-- Удаляем индексы
DROP INDEX IF EXISTS idx_user_state_history_created;
DROP INDEX IF EXISTS idx_user_state_history_telegram;
DROP INDEX IF EXISTS idx_user_state_history_user;
DROP INDEX IF EXISTS idx_state_action_mappings_action;
DROP INDEX IF EXISTS idx_state_action_mappings_state;
DROP INDEX IF EXISTS idx_expected_actions_active;
DROP INDEX IF EXISTS idx_expected_actions_code;
DROP INDEX IF EXISTS idx_user_states_active;
DROP INDEX IF EXISTS idx_user_states_code;

-- Удаляем таблицы
DROP TABLE IF EXISTS user_state_history;
DROP TABLE IF EXISTS state_action_mappings;
DROP TABLE IF EXISTS expected_actions;
DROP TABLE IF EXISTS user_states; 
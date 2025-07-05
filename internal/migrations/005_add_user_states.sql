-- +goose Up

-- Добавляем enum для состояний пользователей
CREATE TYPE user_state AS ENUM (
    'active',           -- Активный пользователь
    'inactive',         -- Неактивный пользователь
    'blocked',          -- Заблокированный пользователь
    'pending_verification', -- Ожидает верификации
    'suspended',        -- Временно приостановлен
    'deleted'           -- Удаленный пользователь
);

-- Добавляем enum для ожидаемых действий
CREATE TYPE expected_action AS ENUM (
    'none',             -- Ничего не ожидается
    'verify_email',     -- Ожидается верификация email
    'complete_profile', -- Ожидается заполнение профиля
    'add_payment',      -- Ожидается добавление платежа
    'contact_support',  -- Ожидается обращение в поддержку
    'wait_approval'     -- Ожидается одобрение администратора
);

-- Добавляем поля состояния в таблицу telegram_users
ALTER TABLE telegram_users 
ADD COLUMN state user_state DEFAULT 'active',
ADD COLUMN expected_action expected_action DEFAULT 'none',
ADD COLUMN state_changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
ADD COLUMN state_reason TEXT,
ADD COLUMN state_changed_by_tg_id BIGINT,
ADD COLUMN state_changed_by_username VARCHAR(255),
ADD COLUMN state_expires_at TIMESTAMP,
ADD COLUMN state_metadata JSONB;

-- Индексы для состояний
CREATE INDEX IF NOT EXISTS idx_telegram_users_state ON telegram_users(state);
CREATE INDEX IF NOT EXISTS idx_telegram_users_expected_action ON telegram_users(expected_action);
CREATE INDEX IF NOT EXISTS idx_telegram_users_state_changed_at ON telegram_users(state_changed_at);
CREATE INDEX IF NOT EXISTS idx_telegram_users_state_expires_at ON telegram_users(state_expires_at);

-- Индекс для поиска по изменению состояния
CREATE INDEX IF NOT EXISTS idx_telegram_users_state_changed_by ON telegram_users(state_changed_by_tg_id);

-- +goose Down

-- Удаляем индексы
DROP INDEX IF EXISTS idx_telegram_users_state_changed_by;
DROP INDEX IF EXISTS idx_telegram_users_state_expires_at;
DROP INDEX IF EXISTS idx_telegram_users_state_changed_at;
DROP INDEX IF EXISTS idx_telegram_users_expected_action;
DROP INDEX IF EXISTS idx_telegram_users_state;

-- Удаляем колонки
ALTER TABLE telegram_users 
DROP COLUMN IF EXISTS state_metadata,
DROP COLUMN IF EXISTS state_expires_at,
DROP COLUMN IF EXISTS state_changed_by_username,
DROP COLUMN IF EXISTS state_changed_by_tg_id,
DROP COLUMN IF EXISTS state_reason,
DROP COLUMN IF EXISTS state_changed_at,
DROP COLUMN IF EXISTS expected_action,
DROP COLUMN IF EXISTS state;

-- Удаляем типы
DROP TYPE IF EXISTS expected_action;
DROP TYPE IF EXISTS user_state; 
-- +goose Up

-- Создаем временные колонки с VARCHAR
ALTER TABLE telegram_users 
ADD COLUMN state_varchar VARCHAR(50) DEFAULT 'active',
ADD COLUMN expected_action_varchar VARCHAR(50) DEFAULT 'none';

-- Копируем данные из enum в VARCHAR
UPDATE telegram_users 
SET state_varchar = state::text,
    expected_action_varchar = expected_action::text;

-- Удаляем старые колонки с enum
ALTER TABLE telegram_users 
DROP COLUMN state,
DROP COLUMN expected_action;

-- Переименовываем новые колонки
ALTER TABLE telegram_users 
RENAME COLUMN state_varchar TO state;

ALTER TABLE telegram_users 
RENAME COLUMN expected_action_varchar TO expected_action;

-- Добавляем внешние ключи для валидации
ALTER TABLE telegram_users 
ADD CONSTRAINT fk_telegram_users_state 
FOREIGN KEY (state) REFERENCES user_states(state_code) ON DELETE RESTRICT;

ALTER TABLE telegram_users 
ADD CONSTRAINT fk_telegram_users_expected_action 
FOREIGN KEY (expected_action) REFERENCES expected_actions(action_code) ON DELETE RESTRICT;

-- Обновляем индексы
DROP INDEX IF EXISTS idx_telegram_users_state;
DROP INDEX IF EXISTS idx_telegram_users_expected_action;

CREATE INDEX IF NOT EXISTS idx_telegram_users_state ON telegram_users(state);
CREATE INDEX IF NOT EXISTS idx_telegram_users_expected_action ON telegram_users(expected_action);

-- +goose Down

-- Удаляем внешние ключи
ALTER TABLE telegram_users 
DROP CONSTRAINT IF EXISTS fk_telegram_users_expected_action;

ALTER TABLE telegram_users 
DROP CONSTRAINT IF EXISTS fk_telegram_users_state;

-- Создаем временные колонки с enum
ALTER TABLE telegram_users 
ADD COLUMN state_enum user_state DEFAULT 'active',
ADD COLUMN expected_action_enum expected_action DEFAULT 'none';

-- Копируем данные обратно в enum (только валидные значения)
UPDATE telegram_users 
SET state_enum = state::user_state
WHERE state IN ('active', 'inactive', 'blocked', 'pending_verification', 'suspended', 'deleted');

UPDATE telegram_users 
SET expected_action_enum = expected_action::expected_action
WHERE expected_action IN ('none', 'verify_email', 'complete_profile', 'add_payment', 'contact_support', 'wait_approval');

-- Удаляем VARCHAR колонки
ALTER TABLE telegram_users 
DROP COLUMN state,
DROP COLUMN expected_action;

-- Переименовываем enum колонки
ALTER TABLE telegram_users 
RENAME COLUMN state_enum TO state;

ALTER TABLE telegram_users 
RENAME COLUMN expected_action_enum TO expected_action;

-- Восстанавливаем индексы
DROP INDEX IF EXISTS idx_telegram_users_state;
DROP INDEX IF EXISTS idx_telegram_users_expected_action;

CREATE INDEX IF NOT EXISTS idx_telegram_users_state ON telegram_users(state);
CREATE INDEX IF NOT EXISTS idx_telegram_users_expected_action ON telegram_users(expected_action); 
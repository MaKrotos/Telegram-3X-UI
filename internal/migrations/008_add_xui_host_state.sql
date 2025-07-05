-- +goose Up

-- Добавляем новое состояние для добавления XUI хоста
INSERT INTO user_states (state_code, state_name, description, can_perform_actions, can_manage_servers, can_create_connections, can_view_only, requires_admin_approval, auto_expire, default_expiry_duration) VALUES
('xui_add_host', 'Добавление XUI хоста', 'Пользователь вводит данные для добавления нового XUI хоста', FALSE, FALSE, FALSE, TRUE, FALSE, TRUE, INTERVAL '10 minutes');

-- Добавляем новое действие для ввода данных хоста
INSERT INTO expected_actions (action_code, action_name, description, priority, auto_resolve, auto_resolve_after) VALUES
('input_host_data', 'Ввод данных хоста', 'Пользователь должен ввести данные хоста в формате: хост логин пароль [секретный_ключ]', 1, FALSE, NULL);

-- Связываем состояние и действие
INSERT INTO state_action_mappings (state_code, action_code, is_default) VALUES
('xui_add_host', 'input_host_data', TRUE);

-- +goose Down

-- Удаляем связь состояния и действия
DELETE FROM state_action_mappings WHERE state_code = 'xui_add_host' AND action_code = 'input_host_data';

-- Удаляем действие
DELETE FROM expected_actions WHERE action_code = 'input_host_data';

-- Удаляем состояние
DELETE FROM user_states WHERE state_code = 'xui_add_host'; 
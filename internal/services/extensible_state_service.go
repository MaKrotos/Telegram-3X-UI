package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// StateDefinition определяет состояние пользователя
type StateDefinition struct {
	ID                    int            `json:"id"`
	StateCode             string         `json:"state_code"`
	StateName             string         `json:"state_name"`
	Description           string         `json:"description"`
	IsActive              bool           `json:"is_active"`
	CanPerformActions     bool           `json:"can_perform_actions"`
	CanManageServers      bool           `json:"can_manage_servers"`
	CanCreateConnections  bool           `json:"can_create_connections"`
	CanViewOnly           bool           `json:"can_view_only"`
	RequiresAdminApproval bool           `json:"requires_admin_approval"`
	AutoExpire            bool           `json:"auto_expire"`
	DefaultExpiryDuration *time.Duration `json:"default_expiry_duration"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

// ActionDefinition определяет ожидаемое действие
type ActionDefinition struct {
	ID               int            `json:"id"`
	ActionCode       string         `json:"action_code"`
	ActionName       string         `json:"action_name"`
	Description      string         `json:"description"`
	IsActive         bool           `json:"is_active"`
	Priority         int            `json:"priority"`
	AutoResolve      bool           `json:"auto_resolve"`
	AutoResolveAfter *time.Duration `json:"auto_resolve_after"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// StateActionMapping связывает состояния и действия
type StateActionMapping struct {
	ID         int       `json:"id"`
	StateCode  string    `json:"state_code"`
	ActionCode string    `json:"action_code"`
	IsDefault  bool      `json:"is_default"`
	CreatedAt  time.Time `json:"created_at"`
}

// UserStateHistory запись истории изменений состояния
type UserStateHistory struct {
	ID                int                    `json:"id"`
	UserID            int                    `json:"user_id"`
	TelegramID        int64                  `json:"telegram_id"`
	OldState          string                 `json:"old_state"`
	NewState          string                 `json:"new_state"`
	OldAction         string                 `json:"old_action"`
	NewAction         string                 `json:"new_action"`
	Reason            string                 `json:"reason"`
	ChangedByTgID     int64                  `json:"changed_by_tg_id"`
	ChangedByUsername string                 `json:"changed_by_username"`
	ExpiresAt         *time.Time             `json:"expires_at"`
	Metadata          map[string]interface{} `json:"metadata"`
	CreatedAt         time.Time              `json:"created_at"`
}

// ExtensibleStateService управляет расширяемой системой состояний
type ExtensibleStateService struct {
	db *sql.DB
}

func NewExtensibleStateService(db *sql.DB) *ExtensibleStateService {
	return &ExtensibleStateService{db: db}
}

// GetStateDefinition получает определение состояния
func (s *ExtensibleStateService) GetStateDefinition(stateCode string) (*StateDefinition, error) {
	query := `
		SELECT id, state_code, state_name, description, is_active,
			   can_perform_actions, can_manage_servers, can_create_connections,
			   can_view_only, requires_admin_approval, auto_expire,
			   default_expiry_duration, created_at, updated_at
		FROM user_states
		WHERE state_code = $1 AND is_active = TRUE
	`

	state := &StateDefinition{}
	var durationStr *string

	err := s.db.QueryRow(query, stateCode).Scan(
		&state.ID, &state.StateCode, &state.StateName, &state.Description, &state.IsActive,
		&state.CanPerformActions, &state.CanManageServers, &state.CanCreateConnections,
		&state.CanViewOnly, &state.RequiresAdminApproval, &state.AutoExpire,
		&durationStr, &state.CreatedAt, &state.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка получения определения состояния: %w", err)
	}

	// Парсим duration
	if durationStr != nil {
		if duration, err := time.ParseDuration(*durationStr); err == nil {
			state.DefaultExpiryDuration = &duration
		}
	}

	return state, nil
}

// GetActionDefinition получает определение действия
func (s *ExtensibleStateService) GetActionDefinition(actionCode string) (*ActionDefinition, error) {
	query := `
		SELECT id, action_code, action_name, description, is_active,
			   priority, auto_resolve, auto_resolve_after, created_at, updated_at
		FROM expected_actions
		WHERE action_code = $1 AND is_active = TRUE
	`

	action := &ActionDefinition{}
	var durationStr *string

	err := s.db.QueryRow(query, actionCode).Scan(
		&action.ID, &action.ActionCode, &action.ActionName, &action.Description, &action.IsActive,
		&action.Priority, &action.AutoResolve, &durationStr, &action.CreatedAt, &action.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка получения определения действия: %w", err)
	}

	// Парсим duration
	if durationStr != nil {
		if duration, err := time.ParseDuration(*durationStr); err == nil {
			action.AutoResolveAfter = &duration
		}
	}

	return action, nil
}

// GetAllStateDefinitions получает все активные определения состояний
func (s *ExtensibleStateService) GetAllStateDefinitions() ([]*StateDefinition, error) {
	query := `
		SELECT id, state_code, state_name, description, is_active,
			   can_perform_actions, can_manage_servers, can_create_connections,
			   can_view_only, requires_admin_approval, auto_expire,
			   default_expiry_duration, created_at, updated_at
		FROM user_states
		WHERE is_active = TRUE
		ORDER BY state_code
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения определений состояний: %w", err)
	}
	defer rows.Close()

	var states []*StateDefinition
	for rows.Next() {
		state := &StateDefinition{}
		var durationStr *string

		err := rows.Scan(
			&state.ID, &state.StateCode, &state.StateName, &state.Description, &state.IsActive,
			&state.CanPerformActions, &state.CanManageServers, &state.CanCreateConnections,
			&state.CanViewOnly, &state.RequiresAdminApproval, &state.AutoExpire,
			&durationStr, &state.CreatedAt, &state.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования определения состояния: %w", err)
		}

		// Парсим duration
		if durationStr != nil {
			if duration, err := time.ParseDuration(*durationStr); err == nil {
				state.DefaultExpiryDuration = &duration
			}
		}

		states = append(states, state)
	}

	return states, nil
}

// GetAllActionDefinitions получает все активные определения действий
func (s *ExtensibleStateService) GetAllActionDefinitions() ([]*ActionDefinition, error) {
	query := `
		SELECT id, action_code, action_name, description, is_active,
			   priority, auto_resolve, auto_resolve_after, created_at, updated_at
		FROM expected_actions
		WHERE is_active = TRUE
		ORDER BY priority, action_code
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения определений действий: %w", err)
	}
	defer rows.Close()

	var actions []*ActionDefinition
	for rows.Next() {
		action := &ActionDefinition{}
		var durationStr *string

		err := rows.Scan(
			&action.ID, &action.ActionCode, &action.ActionName, &action.Description, &action.IsActive,
			&action.Priority, &action.AutoResolve, &durationStr, &action.CreatedAt, &action.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования определения действия: %w", err)
		}

		// Парсим duration
		if durationStr != nil {
			if duration, err := time.ParseDuration(*durationStr); err == nil {
				action.AutoResolveAfter = &duration
			}
		}

		actions = append(actions, action)
	}

	return actions, nil
}

// CreateStateDefinition создает новое определение состояния
func (s *ExtensibleStateService) CreateStateDefinition(state *StateDefinition) error {
	query := `
		INSERT INTO user_states (
			state_code, state_name, description, is_active,
			can_perform_actions, can_manage_servers, can_create_connections,
			can_view_only, requires_admin_approval, auto_expire, default_expiry_duration
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	var durationStr *string
	if state.DefaultExpiryDuration != nil {
		duration := state.DefaultExpiryDuration.String()
		durationStr = &duration
	}

	_, err := s.db.Exec(
		query,
		state.StateCode, state.StateName, state.Description, state.IsActive,
		state.CanPerformActions, state.CanManageServers, state.CanCreateConnections,
		state.CanViewOnly, state.RequiresAdminApproval, state.AutoExpire, durationStr,
	)

	if err != nil {
		return fmt.Errorf("ошибка создания определения состояния: %w", err)
	}

	return nil
}

// CreateActionDefinition создает новое определение действия
func (s *ExtensibleStateService) CreateActionDefinition(action *ActionDefinition) error {
	query := `
		INSERT INTO expected_actions (
			action_code, action_name, description, is_active,
			priority, auto_resolve, auto_resolve_after
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	var durationStr *string
	if action.AutoResolveAfter != nil {
		duration := action.AutoResolveAfter.String()
		durationStr = &duration
	}

	_, err := s.db.Exec(
		query,
		action.ActionCode, action.ActionName, action.Description, action.IsActive,
		action.Priority, action.AutoResolve, durationStr,
	)

	if err != nil {
		return fmt.Errorf("ошибка создания определения действия: %w", err)
	}

	return nil
}

// UpdateStateDefinition обновляет определение состояния
func (s *ExtensibleStateService) UpdateStateDefinition(state *StateDefinition) error {
	query := `
		UPDATE user_states SET
			state_name = $2, description = $3, is_active = $4,
			can_perform_actions = $5, can_manage_servers = $6, can_create_connections = $7,
			can_view_only = $8, requires_admin_approval = $9, auto_expire = $10,
			default_expiry_duration = $11, updated_at = CURRENT_TIMESTAMP
		WHERE state_code = $1
	`

	var durationStr *string
	if state.DefaultExpiryDuration != nil {
		duration := state.DefaultExpiryDuration.String()
		durationStr = &duration
	}

	result, err := s.db.Exec(
		query,
		state.StateCode, state.StateName, state.Description, state.IsActive,
		state.CanPerformActions, state.CanManageServers, state.CanCreateConnections,
		state.CanViewOnly, state.RequiresAdminApproval, state.AutoExpire, durationStr,
	)

	if err != nil {
		return fmt.Errorf("ошибка обновления определения состояния: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества обновленных строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("состояние с кодом %s не найдено", state.StateCode)
	}

	return nil
}

// UpdateActionDefinition обновляет определение действия
func (s *ExtensibleStateService) UpdateActionDefinition(action *ActionDefinition) error {
	query := `
		UPDATE expected_actions SET
			action_name = $2, description = $3, is_active = $4,
			priority = $5, auto_resolve = $6, auto_resolve_after = $7,
			updated_at = CURRENT_TIMESTAMP
		WHERE action_code = $1
	`

	var durationStr *string
	if action.AutoResolveAfter != nil {
		duration := action.AutoResolveAfter.String()
		durationStr = &duration
	}

	result, err := s.db.Exec(
		query,
		action.ActionCode, action.ActionName, action.Description, action.IsActive,
		action.Priority, action.AutoResolve, durationStr,
	)

	if err != nil {
		return fmt.Errorf("ошибка обновления определения действия: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества обновленных строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("действие с кодом %s не найдено", action.ActionCode)
	}

	return nil
}

// DeleteStateDefinition удаляет определение состояния
func (s *ExtensibleStateService) DeleteStateDefinition(stateCode string) error {
	// Проверяем, есть ли пользователи с этим состоянием
	query := `SELECT COUNT(*) FROM telegram_users WHERE state = $1`
	var count int
	err := s.db.QueryRow(query, stateCode).Scan(&count)
	if err != nil {
		return fmt.Errorf("ошибка проверки использования состояния: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("нельзя удалить состояние %s, так как оно используется %d пользователями", stateCode, count)
	}

	// Удаляем состояние
	deleteQuery := `DELETE FROM user_states WHERE state_code = $1`
	result, err := s.db.Exec(deleteQuery, stateCode)
	if err != nil {
		return fmt.Errorf("ошибка удаления определения состояния: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества удаленных строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("состояние с кодом %s не найдено", stateCode)
	}

	return nil
}

// DeleteActionDefinition удаляет определение действия
func (s *ExtensibleStateService) DeleteActionDefinition(actionCode string) error {
	// Проверяем, есть ли пользователи с этим действием
	query := `SELECT COUNT(*) FROM telegram_users WHERE expected_action = $1`
	var count int
	err := s.db.QueryRow(query, actionCode).Scan(&count)
	if err != nil {
		return fmt.Errorf("ошибка проверки использования действия: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("нельзя удалить действие %s, так как оно используется %d пользователями", actionCode, count)
	}

	// Удаляем действие
	deleteQuery := `DELETE FROM expected_actions WHERE action_code = $1`
	result, err := s.db.Exec(deleteQuery, actionCode)
	if err != nil {
		return fmt.Errorf("ошибка удаления определения действия: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества удаленных строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("действие с кодом %s не найдено", actionCode)
	}

	return nil
}

// GetDefaultActionForState получает действие по умолчанию для состояния
func (s *ExtensibleStateService) GetDefaultActionForState(stateCode string) (*ActionDefinition, error) {
	query := `
		SELECT ea.id, ea.action_code, ea.action_name, ea.description, ea.is_active,
			   ea.priority, ea.auto_resolve, ea.auto_resolve_after, ea.created_at, ea.updated_at
		FROM expected_actions ea
		JOIN state_action_mappings sam ON ea.action_code = sam.action_code
		WHERE sam.state_code = $1 AND sam.is_default = TRUE AND ea.is_active = TRUE
	`

	action := &ActionDefinition{}
	var durationStr *string

	err := s.db.QueryRow(query, stateCode).Scan(
		&action.ID, &action.ActionCode, &action.ActionName, &action.Description, &action.IsActive,
		&action.Priority, &action.AutoResolve, &durationStr, &action.CreatedAt, &action.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка получения действия по умолчанию: %w", err)
	}

	// Парсим duration
	if durationStr != nil {
		if duration, err := time.ParseDuration(*durationStr); err == nil {
			action.AutoResolveAfter = &duration
		}
	}

	return action, nil
}

// SetDefaultActionForState устанавливает действие по умолчанию для состояния
func (s *ExtensibleStateService) SetDefaultActionForState(stateCode, actionCode string) error {
	// Сначала сбрасываем все действия по умолчанию для этого состояния
	resetQuery := `UPDATE state_action_mappings SET is_default = FALSE WHERE state_code = $1`
	_, err := s.db.Exec(resetQuery, stateCode)
	if err != nil {
		return fmt.Errorf("ошибка сброса действий по умолчанию: %w", err)
	}

	// Проверяем, существует ли связь
	checkQuery := `SELECT COUNT(*) FROM state_action_mappings WHERE state_code = $1 AND action_code = $2`
	var count int
	err = s.db.QueryRow(checkQuery, stateCode, actionCode).Scan(&count)
	if err != nil {
		return fmt.Errorf("ошибка проверки существования связи: %w", err)
	}

	if count == 0 {
		// Создаем новую связь
		insertQuery := `INSERT INTO state_action_mappings (state_code, action_code, is_default) VALUES ($1, $2, TRUE)`
		_, err = s.db.Exec(insertQuery, stateCode, actionCode)
		if err != nil {
			return fmt.Errorf("ошибка создания связи состояния и действия: %w", err)
		}
	} else {
		// Устанавливаем существующую связь как действие по умолчанию
		updateQuery := `UPDATE state_action_mappings SET is_default = TRUE WHERE state_code = $1 AND action_code = $2`
		_, err = s.db.Exec(updateQuery, stateCode, actionCode)
		if err != nil {
			return fmt.Errorf("ошибка обновления связи состояния и действия: %w", err)
		}
	}

	return nil
}

// AddStateHistory добавляет запись в историю изменений состояний
func (s *ExtensibleStateService) AddStateHistory(history *UserStateHistory) error {
	query := `
		INSERT INTO user_state_history (
			user_id, telegram_id, old_state, new_state, old_action, new_action,
			reason, changed_by_tg_id, changed_by_username, expires_at, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	metadataJSON, err := json.Marshal(history.Metadata)
	if err != nil {
		return fmt.Errorf("ошибка сериализации метаданных: %w", err)
	}

	_, err = s.db.Exec(
		query,
		history.UserID, history.TelegramID, history.OldState, history.NewState,
		history.OldAction, history.NewAction, history.Reason, history.ChangedByTgID,
		history.ChangedByUsername, history.ExpiresAt, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("ошибка добавления записи в историю: %w", err)
	}

	return nil
}

// GetUserStateHistory получает историю изменений состояния пользователя
func (s *ExtensibleStateService) GetUserStateHistory(telegramID int64, limit, offset int) ([]*UserStateHistory, error) {
	query := `
		SELECT id, user_id, telegram_id, old_state, new_state, old_action, new_action,
			   reason, changed_by_tg_id, changed_by_username, expires_at, metadata, created_at
		FROM user_state_history
		WHERE telegram_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Query(query, telegramID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения истории состояний: %w", err)
	}
	defer rows.Close()

	var history []*UserStateHistory
	for rows.Next() {
		record := &UserStateHistory{}
		var metadataJSON []byte

		err := rows.Scan(
			&record.ID, &record.UserID, &record.TelegramID, &record.OldState, &record.NewState,
			&record.OldAction, &record.NewAction, &record.Reason, &record.ChangedByTgID,
			&record.ChangedByUsername, &record.ExpiresAt, &metadataJSON, &record.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования записи истории: %w", err)
		}

		// Парсим JSON метаданные
		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &record.Metadata); err != nil {
				return nil, fmt.Errorf("ошибка парсинга метаданных: %w", err)
			}
		}

		history = append(history, record)
	}

	return history, nil
}

// ValidateStateActionCombination проверяет валидность комбинации состояния и действия
func (s *ExtensibleStateService) ValidateStateActionCombination(stateCode, actionCode string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM state_action_mappings
		WHERE state_code = $1 AND action_code = $2
	`

	var count int
	err := s.db.QueryRow(query, stateCode, actionCode).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки комбинации состояния и действия: %w", err)
	}

	return count > 0, nil
}

// GetAvailableActionsForState получает доступные действия для состояния
func (s *ExtensibleStateService) GetAvailableActionsForState(stateCode string) ([]*ActionDefinition, error) {
	query := `
		SELECT ea.id, ea.action_code, ea.action_name, ea.description, ea.is_active,
			   ea.priority, ea.auto_resolve, ea.auto_resolve_after, ea.created_at, ea.updated_at
		FROM expected_actions ea
		JOIN state_action_mappings sam ON ea.action_code = sam.action_code
		WHERE sam.state_code = $1 AND ea.is_active = TRUE
		ORDER BY sam.is_default DESC, ea.priority, ea.action_code
	`

	rows, err := s.db.Query(query, stateCode)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения доступных действий: %w", err)
	}
	defer rows.Close()

	var actions []*ActionDefinition
	for rows.Next() {
		action := &ActionDefinition{}
		var durationStr *string

		err := rows.Scan(
			&action.ID, &action.ActionCode, &action.ActionName, &action.Description, &action.IsActive,
			&action.Priority, &action.AutoResolve, &durationStr, &action.CreatedAt, &action.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования действия: %w", err)
		}

		// Парсим duration
		if durationStr != nil {
			if duration, err := time.ParseDuration(*durationStr); err == nil {
				action.AutoResolveAfter = &duration
			}
		}

		actions = append(actions, action)
	}

	return actions, nil
}

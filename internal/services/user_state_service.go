package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// UserState представляет состояние пользователя
type UserState string

const (
	UserStateActive              UserState = "active"
	UserStateInactive            UserState = "inactive"
	UserStateBlocked             UserState = "blocked"
	UserStatePendingVerification UserState = "pending_verification"
	UserStateSuspended           UserState = "suspended"
	UserStateDeleted             UserState = "deleted"
)

// ExpectedAction представляет ожидаемое действие от пользователя
type ExpectedAction string

const (
	ExpectedActionNone            ExpectedAction = "none"
	ExpectedActionVerifyEmail     ExpectedAction = "verify_email"
	ExpectedActionCompleteProfile ExpectedAction = "complete_profile"
	ExpectedActionAddPayment      ExpectedAction = "add_payment"
	ExpectedActionContactSupport  ExpectedAction = "contact_support"
	ExpectedActionWaitApproval    ExpectedAction = "wait_approval"
)

// UserStateInfo содержит информацию о состоянии пользователя
type UserStateInfo struct {
	ID                     int                    `json:"id"`
	TelegramID             int64                  `json:"telegram_id"`
	Username               string                 `json:"username"`
	FirstName              string                 `json:"first_name"`
	LastName               string                 `json:"last_name"`
	State                  UserState              `json:"state"`
	ExpectedAction         ExpectedAction         `json:"expected_action"`
	StateChangedAt         time.Time              `json:"state_changed_at"`
	StateReason            string                 `json:"state_reason"`
	StateChangedByTgID     int64                  `json:"state_changed_by_tg_id"`
	StateChangedByUsername string                 `json:"state_changed_by_username"`
	StateExpiresAt         *time.Time             `json:"state_expires_at"`
	StateMetadata          map[string]interface{} `json:"state_metadata"`
	CreatedAt              time.Time              `json:"created_at"`
	UpdatedAt              time.Time              `json:"updated_at"`
	LastActivity           time.Time              `json:"last_activity"`
}

// StateChangeRequest представляет запрос на изменение состояния
type StateChangeRequest struct {
	TelegramID        int64                  `json:"telegram_id"`
	NewState          UserState              `json:"new_state"`
	ExpectedAction    ExpectedAction         `json:"expected_action"`
	Reason            string                 `json:"reason"`
	ChangedByTgID     int64                  `json:"changed_by_tg_id"`
	ChangedByUsername string                 `json:"changed_by_username"`
	ExpiresAt         *time.Time             `json:"expires_at"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// UserStateService управляет состояниями пользователей
type UserStateService struct {
	db *sql.DB
}

func NewUserStateService(db *sql.DB) *UserStateService {
	return &UserStateService{db: db}
}

// GetUserState получает состояние пользователя
func (s *UserStateService) GetUserState(telegramID int64) (*UserStateInfo, error) {
	query := `
		SELECT id, telegram_id, username, first_name, last_name, is_bot,
			   state, expected_action, state_changed_at, state_reason,
			   state_changed_by_tg_id, state_changed_by_username, state_expires_at,
			   state_metadata, created_at, updated_at, last_activity
		FROM telegram_users
		WHERE telegram_id = $1
	`

	user := &UserStateInfo{}
	var stateStr, expectedActionStr string
	var metadataJSON []byte
	var isBot bool

	var stateReason sql.NullString
	var stateChangedByTgID sql.NullInt64
	var stateChangedByUsername sql.NullString
	err := s.db.QueryRow(query, telegramID).Scan(
		&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName, &isBot,
		&stateStr, &expectedActionStr, &user.StateChangedAt, &stateReason,
		&stateChangedByTgID, &stateChangedByUsername, &user.StateExpiresAt,
		&metadataJSON, &user.CreatedAt, &user.UpdatedAt, &user.LastActivity,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка получения состояния пользователя: %w", err)
	}

	user.State = UserState(stateStr)
	user.ExpectedAction = ExpectedAction(expectedActionStr)

	// Обрабатываем NULL значения
	if stateReason.Valid {
		user.StateReason = stateReason.String
	} else {
		user.StateReason = ""
	}

	if stateChangedByTgID.Valid {
		user.StateChangedByTgID = stateChangedByTgID.Int64
	} else {
		user.StateChangedByTgID = 0
	}

	if stateChangedByUsername.Valid {
		user.StateChangedByUsername = stateChangedByUsername.String
	} else {
		user.StateChangedByUsername = ""
	}

	// Парсим JSON метаданные
	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &user.StateMetadata); err != nil {
			return nil, fmt.Errorf("ошибка парсинга метаданных: %w", err)
		}
	}

	return user, nil
}

// UpdateUserState обновляет состояние пользователя
func (s *UserStateService) UpdateUserState(req *StateChangeRequest) error {
	// Проверяем, существует ли пользователь
	existingUser, err := s.GetUserState(req.TelegramID)
	if err != nil {
		return fmt.Errorf("ошибка проверки существования пользователя: %w", err)
	}

	if existingUser == nil {
		return fmt.Errorf("пользователь с Telegram ID %d не найден", req.TelegramID)
	}

	// Подготавливаем метаданные
	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		return fmt.Errorf("ошибка сериализации метаданных: %w", err)
	}

	query := `
		UPDATE telegram_users SET
			state = $2,
			expected_action = $3,
			state_changed_at = CURRENT_TIMESTAMP,
			state_reason = $4,
			state_changed_by_tg_id = $5,
			state_changed_by_username = $6,
			state_expires_at = $7,
			state_metadata = $8,
			updated_at = CURRENT_TIMESTAMP
		WHERE telegram_id = $1
	`

	result, err := s.db.Exec(
		query,
		req.TelegramID, req.NewState, req.ExpectedAction, req.Reason,
		req.ChangedByTgID, req.ChangedByUsername, req.ExpiresAt, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("ошибка обновления состояния пользователя: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества обновленных строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("пользователь с Telegram ID %d не найден", req.TelegramID)
	}

	return nil
}

// GetUsersByState получает пользователей по состоянию
func (s *UserStateService) GetUsersByState(state UserState, limit, offset int) ([]*UserStateInfo, error) {
	query := `
		SELECT id, telegram_id, username, first_name, last_name, is_bot,
			   state, expected_action, state_changed_at, state_reason,
			   state_changed_by_tg_id, state_changed_by_username, state_expires_at,
			   state_metadata, created_at, updated_at, last_activity
		FROM telegram_users
		WHERE state = $1
		ORDER BY state_changed_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Query(query, state, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения пользователей по состоянию: %w", err)
	}
	defer rows.Close()

	var users []*UserStateInfo
	for rows.Next() {
		user := &UserStateInfo{}
		var stateStr, expectedActionStr string
		var metadataJSON []byte
		var isBot bool
		var stateReason sql.NullString

		err := rows.Scan(
			&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName, &isBot,
			&stateStr, &expectedActionStr, &user.StateChangedAt, &stateReason,
			&user.StateChangedByTgID, &user.StateChangedByUsername, &user.StateExpiresAt,
			&metadataJSON, &user.CreatedAt, &user.UpdatedAt, &user.LastActivity,
		)

		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования пользователя: %w", err)
		}

		user.State = UserState(stateStr)
		user.ExpectedAction = ExpectedAction(expectedActionStr)

		// Обрабатываем NULL значения
		if stateReason.Valid {
			user.StateReason = stateReason.String
		} else {
			user.StateReason = ""
		}

		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &user.StateMetadata); err != nil {
				return nil, fmt.Errorf("ошибка парсинга метаданных: %w", err)
			}
		}

		users = append(users, user)
	}

	return users, nil
}

// GetUsersByExpectedAction получает пользователей по ожидаемому действию
func (s *UserStateService) GetUsersByExpectedAction(action ExpectedAction, limit, offset int) ([]*UserStateInfo, error) {
	query := `
		SELECT id, telegram_id, username, first_name, last_name, is_bot,
			   state, expected_action, state_changed_at, state_reason,
			   state_changed_by_tg_id, state_changed_by_username, state_expires_at,
			   state_metadata, created_at, updated_at, last_activity
		FROM telegram_users
		WHERE expected_action = $1
		ORDER BY state_changed_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Query(query, action, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения пользователей по ожидаемому действию: %w", err)
	}
	defer rows.Close()

	var users []*UserStateInfo
	for rows.Next() {
		user := &UserStateInfo{}
		var stateStr, expectedActionStr string
		var metadataJSON []byte
		var isBot bool
		var stateReason sql.NullString

		err := rows.Scan(
			&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName, &isBot,
			&stateStr, &expectedActionStr, &user.StateChangedAt, &stateReason,
			&user.StateChangedByTgID, &user.StateChangedByUsername, &user.StateExpiresAt,
			&metadataJSON, &user.CreatedAt, &user.UpdatedAt, &user.LastActivity,
		)

		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования пользователя: %w", err)
		}

		user.State = UserState(stateStr)
		user.ExpectedAction = ExpectedAction(expectedActionStr)

		// Обрабатываем NULL значения
		if stateReason.Valid {
			user.StateReason = stateReason.String
		} else {
			user.StateReason = ""
		}

		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &user.StateMetadata); err != nil {
				return nil, fmt.Errorf("ошибка парсинга метаданных: %w", err)
			}
		}

		users = append(users, user)
	}

	return users, nil
}

// GetExpiredStates получает пользователей с истекшими состояниями
func (s *UserStateService) GetExpiredStates() ([]*UserStateInfo, error) {
	query := `
		SELECT id, telegram_id, username, first_name, last_name, is_bot,
			   state, expected_action, state_changed_at, state_reason,
			   state_changed_by_tg_id, state_changed_by_username, state_expires_at,
			   state_metadata, created_at, updated_at, last_activity
		FROM telegram_users
		WHERE state_expires_at IS NOT NULL AND state_expires_at < CURRENT_TIMESTAMP
		ORDER BY state_expires_at ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения пользователей с истекшими состояниями: %w", err)
	}
	defer rows.Close()

	var users []*UserStateInfo
	for rows.Next() {
		user := &UserStateInfo{}
		var stateStr, expectedActionStr string
		var metadataJSON []byte
		var isBot bool
		var stateReason sql.NullString

		err := rows.Scan(
			&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName, &isBot,
			&stateStr, &expectedActionStr, &user.StateChangedAt, &stateReason,
			&user.StateChangedByTgID, &user.StateChangedByUsername, &user.StateExpiresAt,
			&metadataJSON, &user.CreatedAt, &user.UpdatedAt, &user.LastActivity,
		)

		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования пользователя: %w", err)
		}

		user.State = UserState(stateStr)
		user.ExpectedAction = ExpectedAction(expectedActionStr)

		// Обрабатываем NULL значения
		if stateReason.Valid {
			user.StateReason = stateReason.String
		} else {
			user.StateReason = ""
		}

		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &user.StateMetadata); err != nil {
				return nil, fmt.Errorf("ошибка парсинга метаданных: %w", err)
			}
		}

		users = append(users, user)
	}

	return users, nil
}

// GetStateStatistics получает статистику по состояниям
func (s *UserStateService) GetStateStatistics() (map[string]interface{}, error) {
	query := `
		SELECT 
			state,
			expected_action,
			COUNT(*) as count
		FROM telegram_users
		GROUP BY state, expected_action
		ORDER BY state, expected_action
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения статистики состояний: %w", err)
	}
	defer rows.Close()

	stats := map[string]interface{}{
		"total_users":  0,
		"by_state":     make(map[string]int),
		"by_action":    make(map[string]int),
		"combinations": make(map[string]int),
	}

	for rows.Next() {
		var stateStr, expectedActionStr string
		var count int

		err := rows.Scan(&stateStr, &expectedActionStr, &count)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования статистики: %w", err)
		}

		// Обновляем общее количество
		stats["total_users"] = stats["total_users"].(int) + count

		// Обновляем статистику по состояниям
		byState := stats["by_state"].(map[string]int)
		byState[stateStr] = byState[stateStr] + count

		// Обновляем статистику по действиям
		byAction := stats["by_action"].(map[string]int)
		byAction[expectedActionStr] = byAction[expectedActionStr] + count

		// Обновляем комбинации
		combinations := stats["combinations"].(map[string]int)
		key := fmt.Sprintf("%s_%s", stateStr, expectedActionStr)
		combinations[key] = count
	}

	return stats, nil
}

// BlockUser блокирует пользователя
func (s *UserStateService) BlockUser(telegramID int64, reason string, changedByTgID int64, changedByUsername string) error {
	req := &StateChangeRequest{
		TelegramID:        telegramID,
		NewState:          UserStateBlocked,
		ExpectedAction:    ExpectedActionContactSupport,
		Reason:            reason,
		ChangedByTgID:     changedByTgID,
		ChangedByUsername: changedByUsername,
		Metadata: map[string]interface{}{
			"blocked_at":   time.Now().Unix(),
			"block_reason": reason,
		},
	}

	return s.UpdateUserState(req)
}

// ActivateUser активирует пользователя
func (s *UserStateService) ActivateUser(telegramID int64, changedByTgID int64, changedByUsername string) error {
	req := &StateChangeRequest{
		TelegramID:        telegramID,
		NewState:          UserStateActive,
		ExpectedAction:    ExpectedActionNone,
		Reason:            "Пользователь активирован",
		ChangedByTgID:     changedByTgID,
		ChangedByUsername: changedByUsername,
		Metadata: map[string]interface{}{
			"activated_at": time.Now().Unix(),
		},
	}

	return s.UpdateUserState(req)
}

// SuspendUser приостанавливает пользователя
func (s *UserStateService) SuspendUser(telegramID int64, reason string, duration time.Duration, changedByTgID int64, changedByUsername string) error {
	expiresAt := time.Now().Add(duration)

	req := &StateChangeRequest{
		TelegramID:        telegramID,
		NewState:          UserStateSuspended,
		ExpectedAction:    ExpectedActionContactSupport,
		Reason:            reason,
		ChangedByTgID:     changedByTgID,
		ChangedByUsername: changedByUsername,
		ExpiresAt:         &expiresAt,
		Metadata: map[string]interface{}{
			"suspended_at":     time.Now().Unix(),
			"suspend_reason":   reason,
			"suspend_duration": duration.String(),
		},
	}

	return s.UpdateUserState(req)
}

// RequestVerification запрашивает верификацию пользователя
func (s *UserStateService) RequestVerification(telegramID int64, reason string, changedByTgID int64, changedByUsername string) error {
	req := &StateChangeRequest{
		TelegramID:        telegramID,
		NewState:          UserStatePendingVerification,
		ExpectedAction:    ExpectedActionVerifyEmail,
		Reason:            reason,
		ChangedByTgID:     changedByTgID,
		ChangedByUsername: changedByUsername,
		Metadata: map[string]interface{}{
			"verification_requested_at": time.Now().Unix(),
			"verification_reason":       reason,
		},
	}

	return s.UpdateUserState(req)
}

// IsUserActive проверяет, активен ли пользователь
func (s *UserStateService) IsUserActive(telegramID int64) (bool, error) {
	user, err := s.GetUserState(telegramID)
	if err != nil {
		return false, err
	}

	if user == nil {
		return false, nil
	}

	return user.State == UserStateActive, nil
}

// CanUserPerformAction проверяет, может ли пользователь выполнять действия
func (s *UserStateService) CanUserPerformAction(telegramID int64) (bool, string, error) {
	user, err := s.GetUserState(telegramID)
	if err != nil {
		return false, "", err
	}

	if user == nil {
		return false, "Пользователь не найден", nil
	}

	switch user.State {
	case UserStateActive:
		return true, "", nil
	case UserStateInactive:
		return false, "Пользователь неактивен", nil
	case UserStateBlocked:
		return false, "Пользователь заблокирован", nil
	case UserStateSuspended:
		if user.StateExpiresAt != nil && time.Now().After(*user.StateExpiresAt) {
			// Приостановка истекла, автоматически активируем
			err := s.ActivateUser(telegramID, 0, "system")
			if err != nil {
				return false, "Ошибка автоматической активации", err
			}
			return true, "", nil
		}
		return false, "Пользователь приостановлен", nil
	case UserStatePendingVerification:
		return false, "Требуется верификация", nil
	case UserStateDeleted:
		return false, "Пользователь удален", nil
	default:
		return false, "Неизвестное состояние", nil
	}
}

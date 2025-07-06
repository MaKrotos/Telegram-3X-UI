package telegram

import (
	"database/sql"
	"fmt"

	"TelegramXUI/internal/contracts"
)

// TelegramUser представляет пользователя в базе данных
type TelegramUser = contracts.TelegramUser

// UserService предоставляет методы для работы с пользователями
type UserService struct {
	db *sql.DB
}

// NewUserService создает новый экземпляр UserService
func NewUserService(db *sql.DB) *UserService {
	return &UserService{db: db}
}

// GetUserByTelegramID получает пользователя по Telegram ID
func (s *UserService) GetUserByTelegramID(telegramID int64) (*contracts.TelegramUser, error) {
	query := `
		SELECT id, telegram_id, username, first_name, last_name, is_bot, 
		       created_at, updated_at, last_activity
		FROM telegram_users 
		WHERE telegram_id = $1
	`

	user := &contracts.TelegramUser{}
	err := s.db.QueryRow(query, telegramID).Scan(
		&user.ID,
		&user.TelegramID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.IsBot,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastActivity,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Пользователь не найден
		}
		return nil, fmt.Errorf("ошибка получения пользователя: %w", err)
	}

	return user, nil
}

// CreateUser создает нового пользователя
func (s *UserService) CreateUser(user *contracts.TelegramUser) error {
	query := `
		INSERT INTO telegram_users (telegram_id, username, first_name, last_name, is_bot)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at, last_activity
	`

	err := s.db.QueryRow(
		query,
		user.TelegramID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.IsBot,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.LastActivity)

	if err != nil {
		return fmt.Errorf("ошибка создания пользователя: %w", err)
	}

	return nil
}

// UpdateUserActivity обновляет время последней активности пользователя
func (s *UserService) UpdateUserActivity(telegramID int64) error {
	query := `
		UPDATE telegram_users 
		SET last_activity = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE telegram_id = $1
	`

	result, err := s.db.Exec(query, telegramID)
	if err != nil {
		return fmt.Errorf("ошибка обновления активности пользователя: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества обновленных строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("пользователь с telegram_id %d не найден", telegramID)
	}

	return nil
}

// EnsureUserExists проверяет существование пользователя и создает его при необходимости
func (s *UserService) EnsureUserExists(telegramUser contracts.User) (*contracts.TelegramUser, error) {
	// Проверяем, существует ли пользователь
	existingUser, err := s.GetUserByTelegramID(int64(telegramUser.ID))
	if err != nil {
		return nil, err
	}

	if existingUser != nil {
		// Пользователь существует, обновляем активность
		if err := s.UpdateUserActivity(int64(telegramUser.ID)); err != nil {
			return nil, err
		}
		return existingUser, nil
	}

	// Пользователь не существует, создаем нового
	newUser := &contracts.TelegramUser{
		TelegramID: int64(telegramUser.ID),
		Username:   telegramUser.Username,
		FirstName:  telegramUser.FirstName,
		LastName:   telegramUser.LastName,
		IsBot:      telegramUser.IsBot,
	}

	if err := s.CreateUser(newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

// GetAllUsers получает всех пользователей
func (s *UserService) GetAllUsers() ([]*contracts.TelegramUser, error) {
	query := `
		SELECT id, telegram_id, username, first_name, last_name, is_bot, 
		       created_at, updated_at, last_activity
		FROM telegram_users 
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения пользователей: %w", err)
	}
	defer rows.Close()

	var users []*contracts.TelegramUser
	for rows.Next() {
		user := &contracts.TelegramUser{}
		err := rows.Scan(
			&user.ID,
			&user.TelegramID,
			&user.Username,
			&user.FirstName,
			&user.LastName,
			&user.IsBot,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.LastActivity,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования пользователя: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по пользователям: %w", err)
	}

	return users, nil
}

// GetUsersCount получает количество пользователей
func (s *UserService) GetUsersCount() (int, error) {
	query := `SELECT COUNT(*) FROM telegram_users`

	var count int
	err := s.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения количества пользователей: %w", err)
	}

	return count, nil
}

// GetDB возвращает подключение к базе данных
func (s *UserService) GetDB() *sql.DB {
	return s.db
}

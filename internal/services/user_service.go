package services

import (
	"database/sql"
	"fmt"
	"log"

	"TelegramXUI/internal/contracts"
)

// UserService предоставляет методы для работы с пользователями
type UserService struct {
	db *sql.DB
}

// NewUserService создает новый сервис пользователей
func NewUserService(db *sql.DB) *UserService {
	return &UserService{
		db: db,
	}
}

// EnsureUserExists проверяет существование пользователя и создает его при необходимости
func (s *UserService) EnsureUserExists(user contracts.User) (*contracts.TelegramUser, error) {
	// Проверяем, существует ли пользователь
	existingUser, err := s.GetUserByTelegramID(int64(user.ID))
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("ошибка проверки существования пользователя: %w", err)
	}

	if existingUser != nil {
		return existingUser, nil
	}

	// Создаем нового пользователя
	newUser := &contracts.TelegramUser{
		TelegramID: int64(user.ID),
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		Username:   user.Username,
		IsBot:      user.IsBot,
	}

	if err := s.CreateUser(newUser); err != nil {
		return nil, fmt.Errorf("ошибка создания пользователя: %w", err)
	}

	log.Printf("[UserService] Создан новый пользователь: %s (ID: %d)", user.FirstName, user.ID)
	return newUser, nil
}

// GetUserByTelegramID получает пользователя по Telegram ID
func (s *UserService) GetUserByTelegramID(telegramID int64) (*contracts.TelegramUser, error) {
	query := `SELECT id, telegram_id, first_name, last_name, username, is_bot, created_at, updated_at 
			  FROM telegram_users WHERE telegram_id = $1`

	var user contracts.TelegramUser
	err := s.db.QueryRow(query, telegramID).Scan(
		&user.ID,
		&user.TelegramID,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.IsBot,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// CreateUser создает нового пользователя
func (s *UserService) CreateUser(user *contracts.TelegramUser) error {
	query := `INSERT INTO telegram_users (telegram_id, first_name, last_name, username, is_bot) 
			  VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`

	return s.db.QueryRow(query, user.TelegramID, user.FirstName, user.LastName, user.Username, user.IsBot).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
}

// GetAllUsers получает всех пользователей
func (s *UserService) GetAllUsers() ([]contracts.TelegramUser, error) {
	query := `SELECT id, telegram_id, first_name, last_name, username, is_bot, created_at, updated_at 
			  FROM telegram_users ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса пользователей: %w", err)
	}
	defer rows.Close()

	var users []contracts.TelegramUser
	for rows.Next() {
		var user contracts.TelegramUser
		err := rows.Scan(
			&user.ID,
			&user.TelegramID,
			&user.FirstName,
			&user.LastName,
			&user.Username,
			&user.IsBot,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования пользователя: %w", err)
		}
		users = append(users, user)
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

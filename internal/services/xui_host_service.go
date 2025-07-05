package services

import (
	"database/sql"
	"fmt"
	"time"
)

type XUIServer struct {
	ID              int       `json:"id"`
	ServerURL       string    `json:"server_url"`
	ServerName      string    `json:"server_name"`
	ServerLocation  string    `json:"server_location"`
	ServerIP        string    `json:"server_ip"`
	ServerPort      int       `json:"server_port"`
	Username        string    `json:"username"`
	Password        string    `json:"password"`
	SecretKey       string    `json:"secret_key"`
	TwoFactorSecret string    `json:"two_factor_secret"`
	IsActive        bool      `json:"is_active"`
	AddedByTgID     int64     `json:"added_by_tg_id"`
	AddedByUsername string    `json:"added_by_username"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type XUIServerService struct {
	db *sql.DB
}

func NewXUIServerService(db *sql.DB) *XUIServerService {
	return &XUIServerService{db: db}
}

// AddServer добавляет новый сервер XUI в базу данных
func (s *XUIServerService) AddServer(server *XUIServer) error {
	query := `
		INSERT INTO xui_servers (
			server_url, server_name, server_location, server_ip, 
			server_port, username, password, secret_key, two_factor_secret, is_active, 
			added_by_tg_id, added_by_username
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRow(
		query,
		server.ServerURL, server.ServerName, server.ServerLocation, server.ServerIP,
		server.ServerPort, server.Username, server.Password, server.SecretKey,
		server.TwoFactorSecret, server.IsActive,
		server.AddedByTgID, server.AddedByUsername,
	).Scan(&server.ID, &server.CreatedAt, &server.UpdatedAt)

	if err != nil {
		return fmt.Errorf("ошибка добавления сервера: %w", err)
	}

	return nil
}

// GetServersByAddedBy получает все серверы, добавленные конкретным пользователем
func (s *XUIServerService) GetServersByAddedBy(tgID int64) ([]*XUIServer, error) {
	query := `
		SELECT id, server_url, server_name, server_location, server_ip,
			   server_port, username, password, secret_key, two_factor_secret, is_active, 
			   added_by_tg_id, added_by_username, created_at, updated_at
		FROM xui_servers
		WHERE added_by_tg_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, tgID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения серверов: %w", err)
	}
	defer rows.Close()

	var servers []*XUIServer
	for rows.Next() {
		server := &XUIServer{}
		err := rows.Scan(
			&server.ID, &server.ServerURL, &server.ServerName, &server.ServerLocation,
			&server.ServerIP, &server.ServerPort, &server.Username, &server.Password,
			&server.SecretKey, &server.TwoFactorSecret, &server.IsActive,
			&server.AddedByTgID, &server.AddedByUsername, &server.CreatedAt, &server.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования сервера: %w", err)
		}
		servers = append(servers, server)
	}

	return servers, nil
}

// GetAllServers получает все серверы с пагинацией
func (s *XUIServerService) GetAllServers(limit, offset int) ([]*XUIServer, error) {
	query := `
		SELECT id, server_url, server_name, server_location, server_ip,
			   server_port, username, password, secret_key, two_factor_secret, is_active,
			   added_by_tg_id, added_by_username, created_at, updated_at
		FROM xui_servers
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения серверов: %w", err)
	}
	defer rows.Close()

	var servers []*XUIServer
	for rows.Next() {
		server := &XUIServer{}
		err := rows.Scan(
			&server.ID, &server.ServerURL, &server.ServerName, &server.ServerLocation,
			&server.ServerIP, &server.ServerPort, &server.Username, &server.Password,
			&server.SecretKey, &server.TwoFactorSecret, &server.IsActive,
			&server.AddedByTgID, &server.AddedByUsername, &server.CreatedAt, &server.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования сервера: %w", err)
		}
		servers = append(servers, server)
	}

	return servers, nil
}

// GetServerByID получает сервер по ID
func (s *XUIServerService) GetServerByID(id int) (*XUIServer, error) {
	query := `
		SELECT id, server_url, server_name, server_location, server_ip,
			   server_port, username, password, secret_key, two_factor_secret, is_active,
			   added_by_tg_id, added_by_username, created_at, updated_at
		FROM xui_servers
		WHERE id = $1
	`

	server := &XUIServer{}
	err := s.db.QueryRow(query, id).Scan(
		&server.ID, &server.ServerURL, &server.ServerName, &server.ServerLocation,
		&server.ServerIP, &server.ServerPort, &server.Username, &server.Password,
		&server.SecretKey, &server.TwoFactorSecret, &server.IsActive,
		&server.AddedByTgID, &server.AddedByUsername, &server.CreatedAt, &server.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка получения сервера: %w", err)
	}

	return server, nil
}

// GetServerByURL получает сервер по URL
func (s *XUIServerService) GetServerByURL(url string) (*XUIServer, error) {
	query := `
		SELECT id, server_url, server_name, server_location, server_ip,
			   server_port, username, password, secret_key, two_factor_secret, is_active,
			   added_by_tg_id, added_by_username, created_at, updated_at
		FROM xui_servers
		WHERE server_url = $1
	`

	server := &XUIServer{}
	err := s.db.QueryRow(query, url).Scan(
		&server.ID, &server.ServerURL, &server.ServerName, &server.ServerLocation,
		&server.ServerIP, &server.ServerPort, &server.Username, &server.Password,
		&server.SecretKey, &server.TwoFactorSecret, &server.IsActive,
		&server.AddedByTgID, &server.AddedByUsername, &server.CreatedAt, &server.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка получения сервера: %w", err)
	}

	return server, nil
}

// UpdateServer обновляет информацию о сервере
func (s *XUIServerService) UpdateServer(server *XUIServer) error {
	query := `
		UPDATE xui_servers SET
			server_url = $2, server_name = $3, server_location = $4,
			server_ip = $5, server_port = $6, username = $7, password = $8,
			secret_key = $9, two_factor_secret = $10, is_active = $11,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING updated_at
	`

	err := s.db.QueryRow(
		query,
		server.ID, server.ServerURL, server.ServerName, server.ServerLocation,
		server.ServerIP, server.ServerPort, server.Username, server.Password,
		server.SecretKey, server.TwoFactorSecret, server.IsActive,
	).Scan(&server.UpdatedAt)

	if err != nil {
		return fmt.Errorf("ошибка обновления сервера: %w", err)
	}

	return nil
}

// DeleteServer удаляет сервер по ID
func (s *XUIServerService) DeleteServer(id int) error {
	query := `DELETE FROM xui_servers WHERE id = $1`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления сервера: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества удаленных строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("сервер с ID %d не найден", id)
	}

	return nil
}

// GetServersCount получает общее количество серверов
func (s *XUIServerService) GetServersCount() (int, error) {
	query := `SELECT COUNT(*) FROM xui_servers`

	var count int
	err := s.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения количества серверов: %w", err)
	}

	return count, nil
}

// GetActiveServers получает только активные серверы
func (s *XUIServerService) GetActiveServers() ([]*XUIServer, error) {
	query := `
		SELECT id, server_url, server_name, server_location, server_ip,
			   server_port, username, password, secret_key, two_factor_secret, is_active,
			   added_by_tg_id, added_by_username, created_at, updated_at
		FROM xui_servers
		WHERE is_active = true
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения активных серверов: %w", err)
	}
	defer rows.Close()

	var servers []*XUIServer
	for rows.Next() {
		server := &XUIServer{}
		err := rows.Scan(
			&server.ID, &server.ServerURL, &server.ServerName, &server.ServerLocation,
			&server.ServerIP, &server.ServerPort, &server.Username, &server.Password,
			&server.SecretKey, &server.TwoFactorSecret, &server.IsActive,
			&server.AddedByTgID, &server.AddedByUsername, &server.CreatedAt, &server.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования сервера: %w", err)
		}
		servers = append(servers, server)
	}

	return servers, nil
}

// GetServersByDateRange получает серверы в определенном диапазоне дат
func (s *XUIServerService) GetServersByDateRange(startDate, endDate time.Time) ([]*XUIServer, error) {
	query := `
		SELECT id, server_url, server_name, server_location, server_ip,
			   server_port, username, password, secret_key, two_factor_secret, is_active,
			   added_by_tg_id, added_by_username, created_at, updated_at
		FROM xui_servers
		WHERE created_at BETWEEN $1 AND $2
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения серверов по датам: %w", err)
	}
	defer rows.Close()

	var servers []*XUIServer
	for rows.Next() {
		server := &XUIServer{}
		err := rows.Scan(
			&server.ID, &server.ServerURL, &server.ServerName, &server.ServerLocation,
			&server.ServerIP, &server.ServerPort, &server.Username, &server.Password,
			&server.SecretKey, &server.TwoFactorSecret, &server.IsActive,
			&server.AddedByTgID, &server.AddedByUsername, &server.CreatedAt, &server.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования сервера: %w", err)
		}
		servers = append(servers, server)
	}

	return servers, nil
}

// SetServerStatus устанавливает статус активности сервера
func (s *XUIServerService) SetServerStatus(serverID int, isActive bool) error {
	query := `
		UPDATE xui_servers SET
			is_active = $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	result, err := s.db.Exec(query, serverID, isActive)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса сервера: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества обновленных строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("сервер с ID %d не найден", serverID)
	}

	return nil
}

// GetInactiveServers получает неактивные серверы
func (s *XUIServerService) GetInactiveServers() ([]*XUIServer, error) {
	query := `
		SELECT id, server_url, server_name, server_location, server_ip,
			   server_port, username, password, secret_key, two_factor_secret, is_active,
			   added_by_tg_id, added_by_username, created_at, updated_at
		FROM xui_servers
		WHERE is_active = false
		ORDER BY updated_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения неактивных серверов: %w", err)
	}
	defer rows.Close()

	var servers []*XUIServer
	for rows.Next() {
		server := &XUIServer{}
		err := rows.Scan(
			&server.ID, &server.ServerURL, &server.ServerName, &server.ServerLocation,
			&server.ServerIP, &server.ServerPort, &server.Username, &server.Password,
			&server.SecretKey, &server.TwoFactorSecret, &server.IsActive,
			&server.AddedByTgID, &server.AddedByUsername, &server.CreatedAt, &server.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования сервера: %w", err)
		}
		servers = append(servers, server)
	}

	return servers, nil
}

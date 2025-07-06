package services

import (
	"database/sql"
	"fmt"
	"time"
)

// VPNConnection представляет VPN подключение пользователя
type VPNConnection struct {
	ID             int       `json:"id"`
	TelegramUserID int64     `json:"telegram_user_id"`
	Username       string    `json:"username"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	ServerID       int       `json:"server_id"`
	InboundID      int       `json:"inbound_id"`
	ClientID       string    `json:"client_id"`
	Email          string    `json:"email"`
	Port           int       `json:"port"`
	VPNLogin       string    `json:"vpn_login"`
	VPNPassword    string    `json:"vpn_password"`
	VlessLink      string    `json:"vless_link"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// VPNConnectionService управляет VPN подключениями
type VPNConnectionService struct {
	db *sql.DB
}

// NewVPNConnectionService создает новый сервис VPN подключений
func NewVPNConnectionService(db *sql.DB) *VPNConnectionService {
	return &VPNConnectionService{db: db}
}

// SaveVPNConnection сохраняет VPN подключение в базу данных
func (s *VPNConnectionService) SaveVPNConnection(connection *VPNConnection) error {
	query := `
		INSERT INTO vpn_connections (
			telegram_user_id, username, first_name, last_name,
			server_id, inbound_id, client_id, email, port,
			vpn_login, vpn_password, vless_link
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRow(
		query,
		connection.TelegramUserID, connection.Username, connection.FirstName, connection.LastName,
		connection.ServerID, connection.InboundID, connection.ClientID, connection.Email, connection.Port,
		connection.VPNLogin, connection.VPNPassword, connection.VlessLink,
	).Scan(&connection.ID, &connection.CreatedAt, &connection.UpdatedAt)

	if err != nil {
		return fmt.Errorf("ошибка сохранения VPN подключения: %w", err)
	}

	return nil
}

// GetUserVPNConnections получает все VPN подключения пользователя
func (s *VPNConnectionService) GetUserVPNConnections(telegramUserID int64) ([]*VPNConnection, error) {
	query := `
		SELECT id, telegram_user_id, username, first_name, last_name,
			   server_id, inbound_id, client_id, email, port,
			   vpn_login, vpn_password, vless_link, is_active,
			   created_at, updated_at
		FROM vpn_connections
		WHERE telegram_user_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, telegramUserID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения VPN подключений: %w", err)
	}
	defer rows.Close()

	var connections []*VPNConnection
	for rows.Next() {
		connection := &VPNConnection{}
		err := rows.Scan(
			&connection.ID, &connection.TelegramUserID, &connection.Username,
			&connection.FirstName, &connection.LastName, &connection.ServerID,
			&connection.InboundID, &connection.ClientID, &connection.Email,
			&connection.Port, &connection.VPNLogin, &connection.VPNPassword,
			&connection.VlessLink, &connection.IsActive, &connection.CreatedAt,
			&connection.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования VPN подключения: %w", err)
		}
		connections = append(connections, connection)
	}

	return connections, nil
}

// GetVPNConnectionByID получает VPN подключение по ID
func (s *VPNConnectionService) GetVPNConnectionByID(id int) (*VPNConnection, error) {
	query := `
		SELECT id, telegram_user_id, username, first_name, last_name,
			   server_id, inbound_id, client_id, email, port,
			   vpn_login, vpn_password, vless_link, is_active,
			   created_at, updated_at
		FROM vpn_connections
		WHERE id = $1
	`

	connection := &VPNConnection{}
	err := s.db.QueryRow(query, id).Scan(
		&connection.ID, &connection.TelegramUserID, &connection.Username,
		&connection.FirstName, &connection.LastName, &connection.ServerID,
		&connection.InboundID, &connection.ClientID, &connection.Email,
		&connection.Port, &connection.VPNLogin, &connection.VPNPassword,
		&connection.VlessLink, &connection.IsActive, &connection.CreatedAt,
		&connection.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка получения VPN подключения: %w", err)
	}

	return connection, nil
}

// DeactivateVPNConnection деактивирует VPN подключение
func (s *VPNConnectionService) DeactivateVPNConnection(id int) error {
	query := `
		UPDATE vpn_connections SET
			is_active = false,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("ошибка деактивации VPN подключения: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества обновленных строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("VPN подключение с ID %d не найдено", id)
	}

	return nil
}

// DeleteVPNConnection удаляет VPN подключение
func (s *VPNConnectionService) DeleteVPNConnection(id int) error {
	query := `DELETE FROM vpn_connections WHERE id = $1`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления VPN подключения: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества удаленных строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("VPN подключение с ID %d не найдено", id)
	}

	return nil
}

// GetUserVPNConnectionsCount получает количество активных VPN подключений пользователя
func (s *VPNConnectionService) GetUserVPNConnectionsCount(telegramUserID int64) (int, error) {
	query := `
		SELECT COUNT(*) FROM vpn_connections
		WHERE telegram_user_id = $1 AND is_active = true
	`

	var count int
	err := s.db.QueryRow(query, telegramUserID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения количества VPN подключений: %w", err)
	}

	return count, nil
}

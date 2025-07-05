package services

import (
	"TelegramXUI/internal/config"
	"fmt"
	"log"
)

type AdminService struct {
	config *config.Config
}

func NewAdminService(cfg *config.Config) *AdminService {
	return &AdminService{
		config: cfg,
	}
}

// IsGlobalAdmin проверяет, является ли пользователь глобальным администратором
func (s *AdminService) IsGlobalAdmin(tgID int64) bool {
	log.Printf("[AdminService] Проверка глобального администратора: tgID=%d, config_tg_id=%d, config_username=%s",
		tgID, s.config.Admin.GlobalAdminTgID, s.config.Admin.GlobalAdminUsername)
	return s.config.Admin.GlobalAdminTgID == tgID
}

// IsGlobalAdminByUsername проверяет, является ли пользователь глобальным администратором по username
func (s *AdminService) IsGlobalAdminByUsername(username string) bool {
	return s.config.Admin.GlobalAdminUsername == username
}

// GetGlobalAdminInfo возвращает информацию о глобальном администраторе
func (s *AdminService) GetGlobalAdminInfo() map[string]interface{} {
	return map[string]interface{}{
		"tg_id":    s.config.Admin.GlobalAdminTgID,
		"username": s.config.Admin.GlobalAdminUsername,
		"is_set":   s.config.Admin.GlobalAdminTgID != 0,
	}
}

// HasAdminPrivileges проверяет, есть ли у пользователя административные привилегии
// Глобальный админ имеет все права
func (s *AdminService) HasAdminPrivileges(tgID int64, username string) bool {
	// Проверяем по Telegram ID
	if s.IsGlobalAdmin(tgID) {
		return true
	}

	// Проверяем по username (если ID не совпал)
	if username != "" && s.IsGlobalAdminByUsername(username) {
		return true
	}

	return false
}

// CanManageServers проверяет, может ли пользователь управлять серверами
func (s *AdminService) CanManageServers(tgID int64, username string) bool {
	return s.HasAdminPrivileges(tgID, username)
}

// CanViewAllServers проверяет, может ли пользователь просматривать все серверы
func (s *AdminService) CanViewAllServers(tgID int64, username string) bool {
	return s.HasAdminPrivileges(tgID, username)
}

// CanDeleteServers проверяет, может ли пользователь удалять серверы
func (s *AdminService) CanDeleteServers(tgID int64, username string) bool {
	return s.HasAdminPrivileges(tgID, username)
}

// CanManageUsers проверяет, может ли пользователь управлять пользователями
func (s *AdminService) CanManageUsers(tgID int64, username string) bool {
	return s.HasAdminPrivileges(tgID, username)
}

// CanViewStats проверяет, может ли пользователь просматривать статистику
func (s *AdminService) CanViewStats(tgID int64, username string) bool {
	return s.HasAdminPrivileges(tgID, username)
}

// GetUserPermissions возвращает все права пользователя
func (s *AdminService) GetUserPermissions(tgID int64, username string) map[string]bool {
	isAdmin := s.HasAdminPrivileges(tgID, username)

	return map[string]bool{
		"is_global_admin":      s.IsGlobalAdmin(tgID) || s.IsGlobalAdminByUsername(username),
		"can_manage_servers":   isAdmin,
		"can_view_all_servers": isAdmin,
		"can_delete_servers":   isAdmin,
		"can_manage_users":     isAdmin,
		"can_view_stats":       isAdmin,
		"can_add_servers":      true, // Все пользователи могут добавлять серверы
		"can_view_own_servers": true, // Все пользователи могут видеть свои серверы
	}
}

// ValidateGlobalAdminConfig проверяет корректность конфигурации глобального админа
func (s *AdminService) ValidateGlobalAdminConfig() error {
	if s.config.Admin.GlobalAdminTgID == 0 {
		return fmt.Errorf("GLOBAL_ADMIN_TG_ID не установлен или равен 0")
	}

	if s.config.Admin.GlobalAdminUsername == "" {
		return fmt.Errorf("GLOBAL_ADMIN_USERNAME не установлен")
	}

	return nil
}

// IsGlobalAdminConfigured проверяет, настроен ли глобальный админ
func (s *AdminService) IsGlobalAdminConfigured() bool {
	return s.config.Admin.GlobalAdminTgID != 0 && s.config.Admin.GlobalAdminUsername != ""
}

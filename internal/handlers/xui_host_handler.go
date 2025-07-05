package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"TelegramXUI/internal/services"
)

type XUIServerHandler struct {
	serverService *services.XUIServerService
	adminService  *services.AdminService
}

func NewXUIServerHandler(serverService *services.XUIServerService, adminService *services.AdminService) *XUIServerHandler {
	return &XUIServerHandler{
		serverService: serverService,
		adminService:  adminService,
	}
}

// GetServersByUser получает все серверы, добавленные конкретным пользователем
func (h *XUIServerHandler) GetServersByUser(w http.ResponseWriter, r *http.Request) {
	tgIDStr := r.URL.Query().Get("tg_id")
	if tgIDStr == "" {
		http.Error(w, "tg_id параметр обязателен", http.StatusBadRequest)
		return
	}

	tgID, err := strconv.ParseInt(tgIDStr, 10, 64)
	if err != nil {
		http.Error(w, "неверный формат tg_id", http.StatusBadRequest)
		return
	}

	// Получаем username из query параметров (если есть)
	username := r.URL.Query().Get("username")

	// Проверяем права доступа
	permissions := h.adminService.GetUserPermissions(tgID, username)

	// Если пользователь не админ, он может видеть только свои серверы
	if !permissions["can_view_all_servers"] {
		// Проверяем, что пользователь запрашивает свои серверы
		requestingTgID, _ := strconv.ParseInt(tgIDStr, 10, 64)
		if requestingTgID != tgID {
			http.Error(w, "недостаточно прав для просмотра серверов других пользователей", http.StatusForbidden)
			return
		}
	}

	servers, err := h.serverService.GetServersByAddedBy(tgID)
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка получения серверов: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":     true,
		"data":        servers,
		"count":       len(servers),
		"permissions": permissions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAllServers получает все серверы с пагинацией (только для админов)
func (h *XUIServerHandler) GetAllServers(w http.ResponseWriter, r *http.Request) {
	// Проверяем права администратора
	tgIDStr := r.URL.Query().Get("admin_tg_id")
	username := r.URL.Query().Get("admin_username")

	if tgIDStr == "" {
		http.Error(w, "admin_tg_id параметр обязателен для просмотра всех серверов", http.StatusBadRequest)
		return
	}

	tgID, err := strconv.ParseInt(tgIDStr, 10, 64)
	if err != nil {
		http.Error(w, "неверный формат admin_tg_id", http.StatusBadRequest)
		return
	}

	// Проверяем права администратора
	if !h.adminService.CanViewAllServers(tgID, username) {
		http.Error(w, "недостаточно прав для просмотра всех серверов", http.StatusForbidden)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // значение по умолчанию
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	servers, err := h.serverService.GetAllServers(limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка получения серверов: %v", err), http.StatusInternalServerError)
		return
	}

	totalCount, err := h.serverService.GetServersCount()
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка получения общего количества: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":     true,
		"data":        servers,
		"count":       len(servers),
		"total_count": totalCount,
		"limit":       limit,
		"offset":      offset,
		"admin_info":  h.adminService.GetGlobalAdminInfo(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetServerByID получает сервер по ID (с проверкой прав)
func (h *XUIServerHandler) GetServerByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "id параметр обязателен", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "неверный формат id", http.StatusBadRequest)
		return
	}

	// Проверяем права доступа
	tgIDStr := r.URL.Query().Get("tg_id")
	username := r.URL.Query().Get("username")

	var tgID int64
	if tgIDStr != "" {
		tgID, _ = strconv.ParseInt(tgIDStr, 10, 64)
	}

	server, err := h.serverService.GetServerByID(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка получения сервера: %v", err), http.StatusInternalServerError)
		return
	}

	if server == nil {
		http.Error(w, "сервер не найден", http.StatusNotFound)
		return
	}

	// Проверяем права доступа к серверу
	if !h.adminService.CanViewAllServers(tgID, username) && server.AddedByTgID != tgID {
		http.Error(w, "недостаточно прав для просмотра этого сервера", http.StatusForbidden)
		return
	}

	response := map[string]interface{}{
		"success":     true,
		"data":        server,
		"permissions": h.adminService.GetUserPermissions(tgID, username),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetServerByURL получает сервер по URL (с проверкой прав)
func (h *XUIServerHandler) GetServerByURL(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "url параметр обязателен", http.StatusBadRequest)
		return
	}

	// Проверяем права доступа
	tgIDStr := r.URL.Query().Get("tg_id")
	username := r.URL.Query().Get("username")

	var tgID int64
	if tgIDStr != "" {
		tgID, _ = strconv.ParseInt(tgIDStr, 10, 64)
	}

	server, err := h.serverService.GetServerByURL(url)
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка получения сервера: %v", err), http.StatusInternalServerError)
		return
	}

	if server == nil {
		http.Error(w, "сервер не найден", http.StatusNotFound)
		return
	}

	// Проверяем права доступа к серверу
	if !h.adminService.CanViewAllServers(tgID, username) && server.AddedByTgID != tgID {
		http.Error(w, "недостаточно прав для просмотра этого сервера", http.StatusForbidden)
		return
	}

	response := map[string]interface{}{
		"success":     true,
		"data":        server,
		"permissions": h.adminService.GetUserPermissions(tgID, username),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetActiveServers получает только активные серверы (с проверкой прав)
func (h *XUIServerHandler) GetActiveServers(w http.ResponseWriter, r *http.Request) {
	// Проверяем права доступа
	tgIDStr := r.URL.Query().Get("tg_id")
	username := r.URL.Query().Get("username")

	var tgID int64
	if tgIDStr != "" {
		tgID, _ = strconv.ParseInt(tgIDStr, 10, 64)
	}

	// Если не админ, возвращаем только его активные серверы
	if !h.adminService.CanViewAllServers(tgID, username) {
		if tgID == 0 {
			http.Error(w, "tg_id параметр обязателен для неадминистраторов", http.StatusBadRequest)
			return
		}

		servers, err := h.serverService.GetServersByAddedBy(tgID)
		if err != nil {
			http.Error(w, fmt.Sprintf("ошибка получения серверов: %v", err), http.StatusInternalServerError)
			return
		}

		// Фильтруем только активные серверы
		var activeServers []*services.XUIServer
		for _, server := range servers {
			if server.IsActive {
				activeServers = append(activeServers, server)
			}
		}

		response := map[string]interface{}{
			"success":  true,
			"data":     activeServers,
			"count":    len(activeServers),
			"filtered": "user_servers_only",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Админ получает все активные серверы
	servers, err := h.serverService.GetActiveServers()
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка получения активных серверов: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":  true,
		"data":     servers,
		"count":    len(servers),
		"filtered": "all_servers",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetServersByDateRange получает серверы в определенном диапазоне дат (только для админов)
func (h *XUIServerHandler) GetServersByDateRange(w http.ResponseWriter, r *http.Request) {
	// Проверяем права администратора
	tgIDStr := r.URL.Query().Get("admin_tg_id")
	username := r.URL.Query().Get("admin_username")

	if tgIDStr == "" {
		http.Error(w, "admin_tg_id параметр обязателен для просмотра серверов по датам", http.StatusBadRequest)
		return
	}

	tgID, err := strconv.ParseInt(tgIDStr, 10, 64)
	if err != nil {
		http.Error(w, "неверный формат admin_tg_id", http.StatusBadRequest)
		return
	}

	if !h.adminService.CanViewStats(tgID, username) {
		http.Error(w, "недостаточно прав для просмотра серверов по датам", http.StatusForbidden)
		return
	}

	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	if startDateStr == "" || endDateStr == "" {
		http.Error(w, "start_date и end_date параметры обязательны", http.StatusBadRequest)
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		http.Error(w, "неверный формат start_date (используйте YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		http.Error(w, "неверный формат end_date (используйте YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	// Устанавливаем время для endDate на конец дня
	endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	servers, err := h.serverService.GetServersByDateRange(startDate, endDate)
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка получения серверов по датам: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":    true,
		"data":       servers,
		"count":      len(servers),
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDateStr,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetServersStats получает статистику по серверам (только для админов)
func (h *XUIServerHandler) GetServersStats(w http.ResponseWriter, r *http.Request) {
	// Проверяем права администратора
	tgIDStr := r.URL.Query().Get("admin_tg_id")
	username := r.URL.Query().Get("admin_username")

	if tgIDStr == "" {
		http.Error(w, "admin_tg_id параметр обязателен для просмотра статистики", http.StatusBadRequest)
		return
	}

	tgID, err := strconv.ParseInt(tgIDStr, 10, 64)
	if err != nil {
		http.Error(w, "неверный формат admin_tg_id", http.StatusBadRequest)
		return
	}

	if !h.adminService.CanViewStats(tgID, username) {
		http.Error(w, "недостаточно прав для просмотра статистики", http.StatusForbidden)
		return
	}

	totalCount, err := h.serverService.GetServersCount()
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка получения статистики: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем серверы за последние 30 дней
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	recentServers, err := h.serverService.GetServersByDateRange(startDate, endDate)
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка получения статистики: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем серверы за последние 7 дней
	weekStartDate := endDate.AddDate(0, 0, -7)
	weekServers, err := h.serverService.GetServersByDateRange(weekStartDate, endDate)
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка получения статистики: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем активные серверы
	activeServers, err := h.serverService.GetActiveServers()
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка получения статистики: %v", err), http.StatusInternalServerError)
		return
	}

	stats := map[string]interface{}{
		"total_servers":        totalCount,
		"active_servers":       len(activeServers),
		"servers_last_30_days": len(recentServers),
		"servers_last_7_days":  len(weekServers),
		"period": map[string]string{
			"start_date": startDate.Format("2006-01-02"),
			"end_date":   endDate.Format("2006-01-02"),
		},
		"admin_info": h.adminService.GetGlobalAdminInfo(),
	}

	response := map[string]interface{}{
		"success": true,
		"data":    stats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAdminInfo возвращает информацию о правах администратора
func (h *XUIServerHandler) GetAdminInfo(w http.ResponseWriter, r *http.Request) {
	tgIDStr := r.URL.Query().Get("tg_id")
	username := r.URL.Query().Get("username")

	var tgID int64
	if tgIDStr != "" {
		tgID, _ = strconv.ParseInt(tgIDStr, 10, 64)
	}

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"permissions":       h.adminService.GetUserPermissions(tgID, username),
			"global_admin_info": h.adminService.GetGlobalAdminInfo(),
			"is_configured":     h.adminService.IsGlobalAdminConfigured(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

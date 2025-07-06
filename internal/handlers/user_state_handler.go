package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"TelegramXUI/internal/services"

	"github.com/gorilla/mux"
)

// UserStateHandler обрабатывает HTTP запросы для управления состояниями пользователей
type UserStateHandler struct {
	userStateService *services.UserStateService
	adminService     *services.AdminService
}

func NewUserStateHandler(userStateService *services.UserStateService, adminService *services.AdminService) *UserStateHandler {
	return &UserStateHandler{
		userStateService: userStateService,
		adminService:     adminService,
	}
}

// GetUserState получает состояние пользователя
func (h *UserStateHandler) GetUserState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	telegramIDStr := vars["telegram_id"]

	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Неверный Telegram ID", http.StatusBadRequest)
		return
	}

	// Проверяем права доступа
	userTgID := getUserTelegramID(r)
	if userTgID == 0 {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	// Пользователь может видеть только свое состояние, админ - любое
	if userTgID != telegramID {
		if !h.adminService.IsGlobalAdmin(userTgID) {
			http.Error(w, "Доступ запрещен", http.StatusForbidden)
			return
		}
	}

	userState, err := h.userStateService.GetUserState(telegramID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка получения состояния: %v", err), http.StatusInternalServerError)
		return
	}

	if userState == nil {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userState)
}

// UpdateUserState обновляет состояние пользователя
func (h *UserStateHandler) UpdateUserState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	telegramIDStr := vars["telegram_id"]

	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Неверный Telegram ID", http.StatusBadRequest)
		return
	}

	// Проверяем права доступа - только админы могут изменять состояния
	userTgID := getUserTelegramID(r)
	if userTgID == 0 {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	if !h.adminService.IsGlobalAdmin(userTgID) {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}

	var req services.StateChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат данных", http.StatusBadRequest)
		return
	}

	req.TelegramID = telegramID
	req.ChangedByTgID = userTgID

	// Получаем username администратора
	adminUser, err := h.userStateService.GetUserState(userTgID)
	if err == nil && adminUser != nil {
		req.ChangedByUsername = adminUser.Username
	}

	if err := h.userStateService.UpdateUserState(&req); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка обновления состояния: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Состояние пользователя обновлено"})
}

// GetUsersByState получает пользователей по состоянию
func (h *UserStateHandler) GetUsersByState(w http.ResponseWriter, r *http.Request) {
	// Проверяем права доступа - только админы
	userTgID := getUserTelegramID(r)
	if userTgID == 0 {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	if !h.adminService.IsGlobalAdmin(userTgID) {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	stateStr := vars["state"]

	state := services.UserState(stateStr)

	// Валидация состояния
	validStates := []services.UserState{
		services.UserStateActive,
		services.UserStateInactive,
		services.UserStateBlocked,
		services.UserStatePendingVerification,
		services.UserStateSuspended,
		services.UserStateDeleted,
	}

	valid := false
	for _, validState := range validStates {
		if state == validState {
			valid = true
			break
		}
	}

	if !valid {
		http.Error(w, "Неверное состояние", http.StatusBadRequest)
		return
	}

	// Параметры пагинации
	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	users, err := h.userStateService.GetUsersByState(state, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка получения пользователей: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// GetUsersByExpectedAction получает пользователей по ожидаемому действию
func (h *UserStateHandler) GetUsersByExpectedAction(w http.ResponseWriter, r *http.Request) {
	// Проверяем права доступа - только админы
	userTgID := getUserTelegramID(r)
	if userTgID == 0 {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	if !h.adminService.IsGlobalAdmin(userTgID) {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	actionStr := vars["action"]

	action := services.ExpectedAction(actionStr)

	// Валидация действия
	validActions := []services.ExpectedAction{
		services.ExpectedActionNone,
		services.ExpectedActionVerifyEmail,
		services.ExpectedActionCompleteProfile,
		services.ExpectedActionAddPayment,
		services.ExpectedActionContactSupport,
		services.ExpectedActionWaitApproval,
	}

	valid := false
	for _, validAction := range validActions {
		if action == validAction {
			valid = true
			break
		}
	}

	if !valid {
		http.Error(w, "Неверное ожидаемое действие", http.StatusBadRequest)
		return
	}

	// Параметры пагинации
	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	users, err := h.userStateService.GetUsersByExpectedAction(action, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка получения пользователей: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// GetExpiredStates получает пользователей с истекшими состояниями
func (h *UserStateHandler) GetExpiredStates(w http.ResponseWriter, r *http.Request) {
	// Проверяем права доступа - только админы
	userTgID := getUserTelegramID(r)
	if userTgID == 0 {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	if !h.adminService.IsGlobalAdmin(userTgID) {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}

	users, err := h.userStateService.GetExpiredStates()
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка получения пользователей с истекшими состояниями: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// GetStateStatistics получает статистику по состояниям
func (h *UserStateHandler) GetStateStatistics(w http.ResponseWriter, r *http.Request) {
	// Проверяем права доступа - только админы
	userTgID := getUserTelegramID(r)
	if userTgID == 0 {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	if !h.adminService.IsGlobalAdmin(userTgID) {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}

	stats, err := h.userStateService.GetStateStatistics()
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка получения статистики: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// BlockUser блокирует пользователя
func (h *UserStateHandler) BlockUser(w http.ResponseWriter, r *http.Request) {
	h.performStateAction(w, r, "block", func(telegramID int64, reason string, changedByTgID int64, changedByUsername string) error {
		return h.userStateService.BlockUser(telegramID, reason, changedByTgID, changedByUsername)
	})
}

// ActivateUser активирует пользователя
func (h *UserStateHandler) ActivateUser(w http.ResponseWriter, r *http.Request) {
	h.performStateAction(w, r, "activate", func(telegramID int64, reason string, changedByTgID int64, changedByUsername string) error {
		return h.userStateService.ActivateUser(telegramID, changedByTgID, changedByUsername)
	})
}

// SuspendUser приостанавливает пользователя
func (h *UserStateHandler) SuspendUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	telegramIDStr := vars["telegram_id"]

	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Неверный Telegram ID", http.StatusBadRequest)
		return
	}

	// Проверяем права доступа - только админы
	userTgID := getUserTelegramID(r)
	if userTgID == 0 {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	if !h.adminService.IsGlobalAdmin(userTgID) {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}

	var req struct {
		Reason   string `json:"reason"`
		Duration string `json:"duration"` // например "24h", "7d"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат данных", http.StatusBadRequest)
		return
	}

	if req.Reason == "" {
		http.Error(w, "Причина приостановки обязательна", http.StatusBadRequest)
		return
	}

	duration, err := time.ParseDuration(req.Duration)
	if err != nil {
		http.Error(w, "Неверный формат длительности", http.StatusBadRequest)
		return
	}

	// Получаем username администратора
	adminUser, err := h.userStateService.GetUserState(userTgID)
	changedByUsername := ""
	if err == nil && adminUser != nil {
		changedByUsername = adminUser.Username
	}

	if err := h.userStateService.SuspendUser(telegramID, req.Reason, duration, userTgID, changedByUsername); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка приостановки пользователя: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Пользователь приостановлен"})
}

// RequestVerification запрашивает верификацию пользователя
func (h *UserStateHandler) RequestVerification(w http.ResponseWriter, r *http.Request) {
	h.performStateAction(w, r, "verify", func(telegramID int64, reason string, changedByTgID int64, changedByUsername string) error {
		return h.userStateService.RequestVerification(telegramID, reason, changedByTgID, changedByUsername)
	})
}

// CheckUserPermission проверяет права пользователя на выполнение действий
func (h *UserStateHandler) CheckUserPermission(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	telegramIDStr := vars["telegram_id"]

	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Неверный Telegram ID", http.StatusBadRequest)
		return
	}

	// Проверяем права доступа
	userTgID := getUserTelegramID(r)
	if userTgID == 0 {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	// Пользователь может проверять только свои права, админ - любые
	if userTgID != telegramID {
		if !h.adminService.IsGlobalAdmin(userTgID) {
			http.Error(w, "Доступ запрещен", http.StatusForbidden)
			return
		}
	}

	canPerform, reason, err := h.userStateService.CanUserPerformAction(telegramID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка проверки прав: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"can_perform": canPerform,
		"reason":      reason,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// performStateAction выполняет действие с состоянием пользователя
func (h *UserStateHandler) performStateAction(w http.ResponseWriter, r *http.Request, actionType string, action func(int64, string, int64, string) error) {
	vars := mux.Vars(r)
	telegramIDStr := vars["telegram_id"]

	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Неверный Telegram ID", http.StatusBadRequest)
		return
	}

	// Проверяем права доступа - только админы
	userTgID := getUserTelegramID(r)
	if userTgID == 0 {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	if !h.adminService.IsGlobalAdmin(userTgID) {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат данных", http.StatusBadRequest)
		return
	}

	// Получаем username администратора
	adminUser, err := h.userStateService.GetUserState(userTgID)
	changedByUsername := ""
	if err == nil && adminUser != nil {
		changedByUsername = adminUser.Username
	}

	if err := action(telegramID, req.Reason, userTgID, changedByUsername); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка выполнения действия: %v", err), http.StatusInternalServerError)
		return
	}

	actionMessages := map[string]string{
		"block":    "Пользователь заблокирован",
		"activate": "Пользователь активирован",
		"verify":   "Запрошена верификация пользователя",
	}

	message := actionMessages[actionType]
	if message == "" {
		message = "Действие выполнено"
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}

// getUserTelegramIDFromRequest извлекает Telegram ID пользователя из запроса
func getUserTelegramIDFromRequest(r *http.Request) int64 {
	// Здесь должна быть логика извлечения Telegram ID из токена или сессии
	// Пока возвращаем 0 для демонстрации
	return 0
}

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"TelegramXUI/internal/services"

	"github.com/gorilla/mux"
)

// ExtensibleStateHandler обрабатывает HTTP запросы для управления расширяемой системой состояний
type ExtensibleStateHandler struct {
	extensibleStateService *services.ExtensibleStateService
	adminService           *services.AdminService
}

func NewExtensibleStateHandler(extensibleStateService *services.ExtensibleStateService, adminService *services.AdminService) *ExtensibleStateHandler {
	return &ExtensibleStateHandler{
		extensibleStateService: extensibleStateService,
		adminService:           adminService,
	}
}

// GetStateDefinitions получает все определения состояний
func (h *ExtensibleStateHandler) GetStateDefinitions(w http.ResponseWriter, r *http.Request) {
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

	states, err := h.extensibleStateService.GetAllStateDefinitions()
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка получения определений состояний: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(states)
}

// GetStateDefinition получает определение конкретного состояния
func (h *ExtensibleStateHandler) GetStateDefinition(w http.ResponseWriter, r *http.Request) {
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
	stateCode := vars["state_code"]

	state, err := h.extensibleStateService.GetStateDefinition(stateCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка получения определения состояния: %v", err), http.StatusInternalServerError)
		return
	}

	if state == nil {
		http.Error(w, "Состояние не найдено", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

// CreateStateDefinition создает новое определение состояния
func (h *ExtensibleStateHandler) CreateStateDefinition(w http.ResponseWriter, r *http.Request) {
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

	var state services.StateDefinition
	if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
		http.Error(w, "Неверный формат данных", http.StatusBadRequest)
		return
	}

	// Валидация обязательных полей
	if state.StateCode == "" {
		http.Error(w, "Код состояния обязателен", http.StatusBadRequest)
		return
	}

	if state.StateName == "" {
		http.Error(w, "Название состояния обязательно", http.StatusBadRequest)
		return
	}

	if err := h.extensibleStateService.CreateStateDefinition(&state); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка создания определения состояния: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Определение состояния создано"})
}

// UpdateStateDefinition обновляет определение состояния
func (h *ExtensibleStateHandler) UpdateStateDefinition(w http.ResponseWriter, r *http.Request) {
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
	stateCode := vars["state_code"]

	var state services.StateDefinition
	if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
		http.Error(w, "Неверный формат данных", http.StatusBadRequest)
		return
	}

	state.StateCode = stateCode

	if err := h.extensibleStateService.UpdateStateDefinition(&state); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка обновления определения состояния: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Определение состояния обновлено"})
}

// DeleteStateDefinition удаляет определение состояния
func (h *ExtensibleStateHandler) DeleteStateDefinition(w http.ResponseWriter, r *http.Request) {
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
	stateCode := vars["state_code"]

	if err := h.extensibleStateService.DeleteStateDefinition(stateCode); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка удаления определения состояния: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Определение состояния удалено"})
}

// GetActionDefinitions получает все определения действий
func (h *ExtensibleStateHandler) GetActionDefinitions(w http.ResponseWriter, r *http.Request) {
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

	actions, err := h.extensibleStateService.GetAllActionDefinitions()
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка получения определений действий: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actions)
}

// GetActionDefinition получает определение конкретного действия
func (h *ExtensibleStateHandler) GetActionDefinition(w http.ResponseWriter, r *http.Request) {
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
	actionCode := vars["action_code"]

	action, err := h.extensibleStateService.GetActionDefinition(actionCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка получения определения действия: %v", err), http.StatusInternalServerError)
		return
	}

	if action == nil {
		http.Error(w, "Действие не найдено", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(action)
}

// CreateActionDefinition создает новое определение действия
func (h *ExtensibleStateHandler) CreateActionDefinition(w http.ResponseWriter, r *http.Request) {
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

	var action services.ActionDefinition
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		http.Error(w, "Неверный формат данных", http.StatusBadRequest)
		return
	}

	// Валидация обязательных полей
	if action.ActionCode == "" {
		http.Error(w, "Код действия обязателен", http.StatusBadRequest)
		return
	}

	if action.ActionName == "" {
		http.Error(w, "Название действия обязательно", http.StatusBadRequest)
		return
	}

	if err := h.extensibleStateService.CreateActionDefinition(&action); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка создания определения действия: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Определение действия создано"})
}

// UpdateActionDefinition обновляет определение действия
func (h *ExtensibleStateHandler) UpdateActionDefinition(w http.ResponseWriter, r *http.Request) {
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
	actionCode := vars["action_code"]

	var action services.ActionDefinition
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		http.Error(w, "Неверный формат данных", http.StatusBadRequest)
		return
	}

	action.ActionCode = actionCode

	if err := h.extensibleStateService.UpdateActionDefinition(&action); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка обновления определения действия: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Определение действия обновлено"})
}

// DeleteActionDefinition удаляет определение действия
func (h *ExtensibleStateHandler) DeleteActionDefinition(w http.ResponseWriter, r *http.Request) {
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
	actionCode := vars["action_code"]

	if err := h.extensibleStateService.DeleteActionDefinition(actionCode); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка удаления определения действия: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Определение действия удалено"})
}

// GetAvailableActionsForState получает доступные действия для состояния
func (h *ExtensibleStateHandler) GetAvailableActionsForState(w http.ResponseWriter, r *http.Request) {
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
	stateCode := vars["state_code"]

	actions, err := h.extensibleStateService.GetAvailableActionsForState(stateCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка получения доступных действий: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actions)
}

// SetDefaultActionForState устанавливает действие по умолчанию для состояния
func (h *ExtensibleStateHandler) SetDefaultActionForState(w http.ResponseWriter, r *http.Request) {
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
	stateCode := vars["state_code"]
	actionCode := vars["action_code"]

	if err := h.extensibleStateService.SetDefaultActionForState(stateCode, actionCode); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка установки действия по умолчанию: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Действие по умолчанию установлено"})
}

// GetUserStateHistory получает историю изменений состояния пользователя
func (h *ExtensibleStateHandler) GetUserStateHistory(w http.ResponseWriter, r *http.Request) {
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

	// Пользователь может видеть только свою историю, админ - любую
	if userTgID != telegramID {
		if !h.adminService.IsGlobalAdmin(userTgID) {
			http.Error(w, "Доступ запрещен", http.StatusForbidden)
			return
		}
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

	history, err := h.extensibleStateService.GetUserStateHistory(telegramID, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка получения истории состояний: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// ValidateStateActionCombination проверяет валидность комбинации состояния и действия
func (h *ExtensibleStateHandler) ValidateStateActionCombination(w http.ResponseWriter, r *http.Request) {
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
	stateCode := vars["state_code"]
	actionCode := vars["action_code"]

	valid, err := h.extensibleStateService.ValidateStateActionCombination(stateCode, actionCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка проверки комбинации: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"valid":       valid,
		"state_code":  stateCode,
		"action_code": actionCode,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetStateManagementInfo получает информацию для управления состояниями
func (h *ExtensibleStateHandler) GetStateManagementInfo(w http.ResponseWriter, r *http.Request) {
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

	// Получаем все состояния и действия
	states, err := h.extensibleStateService.GetAllStateDefinitions()
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка получения состояний: %v", err), http.StatusInternalServerError)
		return
	}

	actions, err := h.extensibleStateService.GetAllActionDefinitions()
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка получения действий: %v", err), http.StatusInternalServerError)
		return
	}

	// Собираем информацию о связях состояний и действий
	stateActionInfo := make(map[string][]string)
	for _, state := range states {
		availableActions, err := h.extensibleStateService.GetAvailableActionsForState(state.StateCode)
		if err != nil {
			continue
		}

		var actionCodes []string
		for _, action := range availableActions {
			actionCodes = append(actionCodes, action.ActionCode)
		}
		stateActionInfo[state.StateCode] = actionCodes
	}

	response := map[string]interface{}{
		"states":                states,
		"actions":               actions,
		"state_action_mappings": stateActionInfo,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getUserTelegramID извлекает Telegram ID пользователя из запроса
func getUserTelegramID(r *http.Request) int64 {
	// Здесь должна быть логика извлечения Telegram ID из токена или сессии
	// Пока возвращаем 0 для демонстрации
	return 0
}

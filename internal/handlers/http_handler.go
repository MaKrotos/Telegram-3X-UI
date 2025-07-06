package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"TelegramXUI/internal/services"
)

// User представляет пользователя для API
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// HTTPHandler обрабатывает HTTP запросы
type HTTPHandler struct {
	userService *services.UserService
	vpnService  *services.VPNService
	webAppURL   string
}

// NewHTTPHandler создает новый HTTP обработчик
func NewHTTPHandler(userService *services.UserService, vpnService *services.VPNService, webAppURL string) *HTTPHandler {
	return &HTTPHandler{
		userService: userService,
		vpnService:  vpnService,
		webAppURL:   webAppURL,
	}
}

// GetUsersHandler обрабатывает запрос на получение пользователей
func (h *HTTPHandler) GetUsersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Попытка подключения к x-ui
		var xuiStatus string
		if h.vpnService != nil {
			if err := h.vpnService.CheckStatus(); err != nil {
				xuiStatus = "Ошибка подключения к x-ui: " + err.Error()
			} else {
				xuiStatus = "Успешное подключение к x-ui"
			}
		} else {
			xuiStatus = "x-ui клиент не инициализирован"
		}

		// Получаем пользователей из базы данных
		var users []User
		if h.userService != nil {
			telegramUsers, err := h.userService.GetAllUsers()
			if err != nil {
				log.Printf("Ошибка получения пользователей: %v", err)
			} else {
				for _, tu := range telegramUsers {
					users = append(users, User{
						ID:   int(tu.TelegramID),
						Name: tu.FirstName + " " + tu.LastName,
					})
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"xui_status": xuiStatus,
			"users":      users,
			"webapp_url": h.webAppURL,
		})
	}
}

// GetTelegramUsersHandler обрабатывает запрос на получение Telegram пользователей
func (h *HTTPHandler) GetTelegramUsersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.userService == nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Сервис пользователей не инициализирован",
			})
			return
		}

		users, err := h.userService.GetAllUsers()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Ошибка получения пользователей: " + err.Error(),
			})
			return
		}

		count, err := h.userService.GetUsersCount()
		if err != nil {
			log.Printf("Ошибка получения количества пользователей: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"users":       users,
			"total_count": count,
			"webapp_url":  h.webAppURL,
		})
	}
}

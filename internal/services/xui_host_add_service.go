package services

import (
	"TelegramXUI/internal/xui_client"
	"fmt"
	"strings"
	"time"
)

// XUIHostData содержит данные для добавления XUI хоста
type XUIHostData struct {
	Host      string `json:"host"`
	Login     string `json:"login"`
	Password  string `json:"password"`
	SecretKey string `json:"secret_key,omitempty"`
}

// XUIHostAddService управляет процессом добавления XUI хостов через Telegram
type XUIHostAddService struct {
	userStateService       *UserStateService
	extensibleStateService *ExtensibleStateService
	xuiServerService       *XUIServerService
	adminService           *AdminService
}

func NewXUIHostAddService(
	userStateService *UserStateService,
	extensibleStateService *ExtensibleStateService,
	xuiServerService *XUIServerService,
	adminService *AdminService,
) *XUIHostAddService {
	return &XUIHostAddService{
		userStateService:       userStateService,
		extensibleStateService: extensibleStateService,
		xuiServerService:       xuiServerService,
		adminService:           adminService,
	}
}

// StartAddHostProcess начинает процесс добавления хоста
func (s *XUIHostAddService) StartAddHostProcess(telegramID int64, username string) error {
	// Проверяем, является ли пользователь глобальным админом
	if !s.adminService.IsGlobalAdmin(telegramID) {
		return fmt.Errorf("только глобальные администраторы могут добавлять XUI хосты")
	}
	// Переводим пользователя в состояние добавления хоста
	expiresAt := time.Now().Add(10 * time.Minute)
	req := &StateChangeRequest{
		TelegramID:        telegramID,
		NewState:          "xui_add_host",
		ExpectedAction:    "input_host_data",
		Reason:            "Начало процесса добавления XUI хоста",
		ChangedByTgID:     telegramID,
		ChangedByUsername: username,
		ExpiresAt:         &expiresAt, // 10 минут на ввод
		Metadata: map[string]interface{}{
			"process_started_at": time.Now().Unix(),
			"process_type":       "xui_host_add",
		},
	}
	if err := s.userStateService.UpdateUserState(req); err != nil {
		return fmt.Errorf("ошибка перевода в состояние добавления хоста: %w", err)
	}
	return nil
}

// ProcessHostData обрабатывает введенные данные хоста
func (s *XUIHostAddService) ProcessHostData(telegramID int64, message string, username string) (*XUIHostData, error) {
	// Проверяем, что пользователь находится в правильном состоянии
	userState, err := s.userStateService.GetUserState(telegramID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения состояния пользователя: %w", err)
	}

	if userState == nil {
		return nil, fmt.Errorf("пользователь не найден")
	}

	if userState.State != "xui_add_host" {
		return nil, fmt.Errorf("пользователь не находится в состоянии добавления хоста")
	}

	// Парсим данные хоста
	hostData, err := s.parseHostData(message)
	if err != nil {
		return nil, err
	}

	// Проверяем подключение к хосту
	if err := s.testHostConnection(hostData); err != nil {
		return nil, fmt.Errorf("ошибка подключения к хосту: %w", err)
	}

	// Добавляем хост в базу данных
	server := &XUIServer{
		ServerURL:       hostData.Host,
		ServerName:      fmt.Sprintf("XUI Server - %s", hostData.Host),
		ServerLocation:  "", // Можно доработать геолокацию
		ServerIP:        s.extractIPFromHost(hostData.Host),
		ServerPort:      s.extractPortFromHost(hostData.Host),
		Username:        hostData.Login,
		Password:        hostData.Password,
		SecretKey:       hostData.SecretKey,
		TwoFactorSecret: "",
		IsActive:        true,
		AddedByTgID:     telegramID,
		AddedByUsername: username,
	}

	if err := s.xuiServerService.AddServer(server); err != nil {
		return nil, fmt.Errorf("ошибка сохранения хоста в базу данных: %w", err)
	}

	// Возвращаем пользователя в активное состояние
	activateReq := &StateChangeRequest{
		TelegramID:        telegramID,
		NewState:          "active",
		ExpectedAction:    "none",
		Reason:            "XUI хост успешно добавлен",
		ChangedByTgID:     telegramID,
		ChangedByUsername: username,
		Metadata: map[string]interface{}{
			"host_added": hostData.Host,
			"added_at":   time.Now().Unix(),
		},
	}

	if err := s.userStateService.UpdateUserState(activateReq); err != nil {
		return nil, fmt.Errorf("ошибка возврата в активное состояние: %w", err)
	}

	return hostData, nil
}

// parseHostData парсит данные хоста из сообщения
func (s *XUIHostAddService) parseHostData(message string) (*XUIHostData, error) {
	// Убираем лишние пробелы и разбиваем на части
	parts := strings.Fields(strings.TrimSpace(message))

	// Попытка автокоррекции и догадки по формату
	normalized, changed := normalizeHostInput(parts)
	if changed {
		return nil, fmt.Errorf("Похоже, вы имели в виду: %s\n\nЕсли всё верно, отправьте это сообщение повторно. Если нет — исправьте ввод по примеру: host login password [secret]", normalized)
	}

	parts = strings.Fields(strings.TrimSpace(normalized))

	if len(parts) < 3 {
		return nil, fmt.Errorf("Недостаточно данных. Формат: хост логин пароль [секретный_ключ]")
	}

	if len(parts) > 4 {
		return nil, fmt.Errorf("Слишком много данных. Формат: хост логин пароль [секретный_ключ]")
	}

	hostData := &XUIHostData{
		Host:     parts[0],
		Login:    parts[1],
		Password: parts[2],
	}

	// Секретный ключ опциональный
	if len(parts) == 4 {
		hostData.SecretKey = parts[3]
	}

	// Валидация данных
	if err := s.validateHostData(hostData); err != nil {
		return nil, err
	}

	return hostData, nil
}

// normalizeHostInput пытается догадаться о правильном формате и вернуть исправленный вариант
func normalizeHostInput(parts []string) (string, bool) {
	if len(parts) == 1 {
		// Возможно, пользователь ввёл всё через двоеточие или пробелы не там
		p := parts[0]
		p = strings.ReplaceAll(p, ",", " ")
		p = strings.ReplaceAll(p, ";", " ")
		p = strings.ReplaceAll(p, "://", "__PROTOCOL__")
		p = strings.ReplaceAll(p, ":", " ")
		p = strings.ReplaceAll(p, "__PROTOCOL__", "://")
		p = strings.ReplaceAll(p, "  ", " ")
		p = strings.TrimSpace(p)
		return p, true
	}
	if len(parts) == 2 {
		// Возможно, забыли пароль
		return parts[0] + " " + parts[1] + " <пароль>", true
	}
	if len(parts) == 3 || len(parts) == 4 {
		// Проверим, начинается ли host с http/https
		host := parts[0]
		if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
			if strings.Contains(host, ":") {
				host = "http://" + host
				return host + " " + strings.Join(parts[1:], " "), true
			}
		}
	}
	return strings.Join(parts, " "), false
}

// validateHostData валидирует данные хоста
func (s *XUIHostAddService) validateHostData(data *XUIHostData) error {
	if data.Host == "" {
		return fmt.Errorf("хост не может быть пустым")
	}

	if data.Login == "" {
		return fmt.Errorf("логин не может быть пустым")
	}

	if data.Password == "" {
		return fmt.Errorf("пароль не может быть пустым")
	}

	// Проверяем формат хоста (должен содержать http:// или https://)
	if !strings.HasPrefix(data.Host, "http://") && !strings.HasPrefix(data.Host, "https://") {
		return fmt.Errorf("хост должен начинаться с http:// или https://")
	}

	return nil
}

// testHostConnection тестирует подключение к хосту
func (s *XUIHostAddService) testHostConnection(data *XUIHostData) error {
	// Создаем временный клиент для тестирования
	client := xui_client.NewClient(data.Host, data.Login, data.Password)

	// Пытаемся авторизоваться на сервере
	if err := client.Login(); err != nil {
		return fmt.Errorf("не удалось подключиться к XUI серверу: %v", err)
	}

	return nil
}

// extractIPFromHost извлекает IP из URL хоста
func (s *XUIHostAddService) extractIPFromHost(host string) string {
	// Убираем протокол
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")

	// Убираем порт если есть
	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	// Убираем путь если есть
	if slashIndex := strings.Index(host, "/"); slashIndex != -1 {
		host = host[:slashIndex]
	}

	return host
}

// extractPortFromHost извлекает порт из URL хоста
func (s *XUIHostAddService) extractPortFromHost(host string) int {
	// Убираем протокол
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")

	// Ищем порт
	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		host = host[colonIndex+1:]
		// Убираем путь если есть
		if slashIndex := strings.Index(host, "/"); slashIndex != -1 {
			host = host[:slashIndex]
		}

		// Пытаемся преобразовать в число
		if port, err := fmt.Sscanf(host, "%d", new(int)); err == nil && port == 1 {
			var portNum int
			fmt.Sscanf(host, "%d", &portNum)
			return portNum
		}
	}

	// Возвращаем порт по умолчанию для HTTPS
	if strings.HasPrefix(host, "https://") {
		return 443
	}
	return 80
}

// CancelAddHostProcess отменяет процесс добавления хоста
func (s *XUIHostAddService) CancelAddHostProcess(telegramID int64, username string) error {
	// Проверяем, что пользователь находится в правильном состоянии
	userState, err := s.userStateService.GetUserState(telegramID)
	if err != nil {
		return fmt.Errorf("ошибка получения состояния пользователя: %w", err)
	}

	if userState == nil {
		return fmt.Errorf("пользователь не найден")
	}

	if userState.State != "xui_add_host" {
		return fmt.Errorf("пользователь не находится в состоянии добавления хоста")
	}

	// Возвращаем пользователя в активное состояние
	activateReq := &StateChangeRequest{
		TelegramID:        telegramID,
		NewState:          "active",
		ExpectedAction:    "none",
		Reason:            "Процесс добавления хоста отменен",
		ChangedByTgID:     telegramID,
		ChangedByUsername: username,
		Metadata: map[string]interface{}{
			"process_cancelled_at": time.Now().Unix(),
		},
	}

	if err := s.userStateService.UpdateUserState(activateReq); err != nil {
		return fmt.Errorf("ошибка возврата в активное состояние: %w", err)
	}

	return nil
}

// GetAddHostInstructions возвращает инструкции для добавления хоста
func (s *XUIHostAddService) GetAddHostInstructions() string {
	return `📝 Инструкция по добавлению XUI хоста:

Введите данные в следующем формате:
хост логин пароль [секретный_ключ]

Примеры:
• https://example.com admin password123
• http://192.168.1.100:54321 user pass 2fa_secret

Поля:
• хост - URL XUI сервера (обязательно)
• логин - имя пользователя (обязательно)
• пароль - пароль (обязательно)
• секретный_ключ - ключ 2FA (необязательно)

⏰ У вас есть 10 минут на ввод данных.
❌ Для отмены отправьте /cancel`
}

// IsInAddHostState проверяет, находится ли пользователь в состоянии добавления хоста
func (s *XUIHostAddService) IsInAddHostState(telegramID int64) (bool, error) {
	userState, err := s.userStateService.GetUserState(telegramID)
	if err != nil {
		return false, err
	}

	if userState == nil {
		return false, nil
	}

	return userState.State == "xui_add_host", nil
}

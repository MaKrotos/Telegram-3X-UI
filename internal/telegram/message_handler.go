package telegram

import (
	"TelegramXUI/internal/contracts"
	"TelegramXUI/internal/services"
	"TelegramXUI/internal/xui_client"
	"fmt"
	"log"
	"strings"
	"time"
)

// MessageProcessor обрабатывает сообщения Telegram бота
type MessageProcessor struct {
	userStateService       contracts.UserStateService
	extensibleStateService contracts.ExtensibleStateService
	xuiHostAddService      contracts.XUIHostAddService
	adminService           contracts.AdminService
	xuiServerService       *services.XUIServerService
	hostMonitorService     *services.HostMonitorService
}

// NewMessageProcessor создает новый обработчик сообщений
func NewMessageProcessor(
	userStateService contracts.UserStateService,
	extensibleStateService contracts.ExtensibleStateService,
	xuiHostAddService contracts.XUIHostAddService,
	adminService contracts.AdminService,
	xuiServerService *services.XUIServerService,
	hostMonitorService *services.HostMonitorService,
) *MessageProcessor {
	return &MessageProcessor{
		userStateService:       userStateService,
		extensibleStateService: extensibleStateService,
		xuiHostAddService:      xuiHostAddService,
		adminService:           adminService,
		xuiServerService:       xuiServerService,
		hostMonitorService:     hostMonitorService,
	}
}

// ProcessMessage обрабатывает входящее сообщение
func (p *MessageProcessor) ProcessMessage(client *TelegramClient, update Update) error {
	// Обрабатываем только текстовые сообщения и callback queries
	if update.Message == nil && update.CallbackQuery == nil {
		return nil
	}

	var userID int64
	var username string
	var messageText string
	var chatID int

	// Извлекаем данные пользователя
	if update.Message != nil {
		userID = int64(update.Message.From.ID)
		username = update.Message.From.Username
		messageText = update.Message.Text
		chatID = update.Message.Chat.ID
	} else if update.CallbackQuery != nil {
		userID = int64(update.CallbackQuery.From.ID)
		username = update.CallbackQuery.From.Username
		messageText = update.CallbackQuery.Data
		chatID = int(userID)

		// Обрабатываем callback-запросы
		if messageText == "addhost" {
			// Получаем состояние пользователя для callback-запроса
			userState, err := p.userStateService.GetUserState(userID)
			if err != nil {
				log.Printf("[MessageProcessor] Ошибка получения состояния пользователя %d: %v", userID, err)
				return p.sendErrorMessage(client, chatID, "Ошибка получения состояния пользователя")
			}

			// Отвечаем на callback-запрос
			if _, err := client.AnswerCallbackQuery(update.CallbackQuery.ID, ""); err != nil {
				log.Printf("[MessageProcessor] Ошибка ответа на callback-запрос: %v", err)
			}

			return p.handleAddHostCommand(client, chatID, userID, username, userState)
		} else if messageText == "create_vpn" {
			if _, err := client.AnswerCallbackQuery(update.CallbackQuery.ID, ""); err != nil {
				log.Printf("[MessageProcessor] Ошибка ответа на callback create_vpn: %v", err)
			}
			return p.handleCreateVPN(client, chatID, userID)
		}
	}

	// Проверяем, что у нас есть текст сообщения
	if messageText == "" {
		return nil
	}

	log.Printf("[MessageProcessor] Обработка сообщения от пользователя %d (%s): %s", userID, username, messageText)

	// Проверяем состояние пользователя
	userState, err := p.userStateService.GetUserState(userID)
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка получения состояния пользователя %d: %v", userID, err)
		return p.sendErrorMessage(client, chatID, "Ошибка получения состояния пользователя")
	}

	// Если пользователь не найден, создаем его с активным состоянием
	if userState == nil {
		log.Printf("[MessageProcessor] Пользователь %d не найден, создаем активное состояние", userID)
		// Здесь должна быть логика создания пользователя
		// Пока просто отправляем сообщение об ошибке
		return p.sendErrorMessage(client, chatID, "Пользователь не зарегистрирован в системе")
	}

	// Обрабатываем команды
	if strings.HasPrefix(messageText, "/") {
		return p.handleCommand(client, chatID, userID, username, messageText, userState)
	}

	// Обрабатываем сообщения в зависимости от состояния
	switch userState.State {
	case "xui_add_host":
		return p.handleAddHostMessage(client, chatID, userID, username, messageText)
	default:
		return p.handleDefaultMessage(client, chatID, userID, username, messageText, userState)
	}
}

// handleCommand обрабатывает команды
func (p *MessageProcessor) handleCommand(client *TelegramClient, chatID int, userID int64, username, command string, userState *contracts.UserStateInfo) error {
	command = strings.ToLower(strings.TrimSpace(command))

	log.Printf("[MessageProcessor] Обработка команды: %s от пользователя %d (%s)", command, userID, username)
	log.Printf("[MessageProcessor] Проверка прав администратора для пользователя %d: %v", userID, p.adminService.IsGlobalAdmin(userID))

	switch command {
	case "/start":
		return p.handleStartCommand(client, chatID, userID, username, userState)
	case "/help":
		return p.handleHelpCommand(client, chatID, userID, username, userState)
	case "/cancel":
		return p.handleCancelCommand(client, chatID, userID, username, userState)
	case "/addhost":
		return p.handleAddHostCommand(client, chatID, userID, username, userState)
	case "/monitor":
		return p.handleMonitorCommand(client, chatID, userID, username, userState)
	case "/monitor_start":
		return p.handleMonitorStartCommand(client, chatID, userID, username, userState)
	case "/monitor_stop":
		return p.handleMonitorStopCommand(client, chatID, userID, username, userState)
	case "/monitor_status":
		return p.handleMonitorStatusCommand(client, chatID, userID, username, userState)
	case "/check_hosts":
		return p.handleCheckHostsCommand(client, chatID, userID, username, userState)
	default:
		return p.sendMessage(client, chatID, fmt.Sprintf("Неизвестная команда: %s\nИспользуйте /help для получения справки", command))
	}
}

// handleStartCommand обрабатывает команду /start
func (p *MessageProcessor) handleStartCommand(client *TelegramClient, chatID int, userID int64, username string, userState *contracts.UserStateInfo) error {
	message := fmt.Sprintf(`🤖 **Добро пожаловать в TelegramXUI!**

👤 **Пользователь:** %s
📊 **Состояние:** %s

Используйте /help для получения справки по командам.`, username, userState.State)
	if p.adminService.IsGlobalAdmin(userID) {
		return p.sendMessageHTML(client, chatID, message)
	}
	return p.sendMessageWithCreateVPNButton(client, chatID, message)
}

// handleHelpCommand обрабатывает команду /help
func (p *MessageProcessor) handleHelpCommand(client *TelegramClient, chatID int, userID int64, username string, userState *contracts.UserStateInfo) error {
	message := `📚 **Справка по командам:**

/start - Начать работу с ботом
/help - Показать эту справку
/cancel - Отменить текущую операцию`

	if p.adminService.IsGlobalAdmin(userID) {
		message += `\n\n🔧 **Команды администратора:**
/addhost - Добавить XUI хост
/monitor - Управление мониторингом хостов
/monitor_start - Запустить мониторинг
/monitor_stop - Остановить мониторинг
/monitor_status - Статус мониторинга
/check_hosts - Проверить все хосты сейчас`
		return p.sendMessageWithInlineAddHost(client, chatID, message)
	}
	// Для обычных пользователей добавляем кнопку "Создать VPN"
	return p.sendMessageWithCreateVPNButton(client, chatID, message)
}

// handleCancelCommand обрабатывает команду /cancel
func (p *MessageProcessor) handleCancelCommand(client *TelegramClient, chatID int, userID int64, username string, userState *contracts.UserStateInfo) error {
	// Проверяем, находится ли пользователь в состоянии добавления хоста
	isInAddHostState, err := p.xuiHostAddService.IsInAddHostState(userID)
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка проверки состояния добавления хоста: %v", err)
		return p.sendErrorMessage(client, chatID, "Ошибка проверки состояния")
	}

	if isInAddHostState {
		// Отменяем процесс добавления хоста
		if err := p.xuiHostAddService.CancelAddHostProcess(userID, username); err != nil {
			log.Printf("[MessageProcessor] Ошибка отмены процесса добавления хоста: %v", err)
			return p.sendErrorMessage(client, chatID, "Ошибка отмены процесса")
		}

		return p.sendMessage(client, chatID, "✅ Процесс добавления хоста отменен. Вы вернулись в обычное состояние.")
	}

	return p.sendMessage(client, chatID, "❌ Нет активного процесса для отмены.")
}

// handleAddHostCommand обрабатывает команду /addhost
func (p *MessageProcessor) handleAddHostCommand(client *TelegramClient, chatID int, userID int64, username string, userState *contracts.UserStateInfo) error {
	// Проверяем, является ли пользователь глобальным админом
	if !p.adminService.IsGlobalAdmin(userID) {
		return p.sendMessage(client, chatID, "❌ Только глобальные администраторы могут добавлять XUI хосты.")
	}

	// Начинаем процесс добавления хоста
	if err := p.xuiHostAddService.StartAddHostProcess(userID, username); err != nil {
		log.Printf("[MessageProcessor] Ошибка начала процесса добавления хоста: %v", err)
		return p.sendErrorMessage(client, chatID, fmt.Sprintf("Ошибка начала процесса: %v", err))
	}

	// Отправляем инструкции
	instructions := p.xuiHostAddService.GetAddHostInstructions()
	return p.sendMessageHTML(client, chatID, instructions)
}

// handleAddHostMessage обрабатывает сообщения в состоянии добавления хоста
func (p *MessageProcessor) handleAddHostMessage(client *TelegramClient, chatID int, userID int64, username, messageText string) error {
	// Обрабатываем данные хоста
	hostData, err := p.xuiHostAddService.ProcessHostData(userID, messageText, username)
	if err != nil {
		// Если ошибка, отправляем сообщение об ошибке и ждем повторного ввода
		errorMessage := fmt.Sprintf("❌ **Ошибка:** %v\n\nПопробуйте еще раз или отправьте /cancel для отмены.", err)
		return p.sendMessageHTML(client, chatID, errorMessage)
	}

	// Успешно добавлен хост
	successMessage := fmt.Sprintf("✅ **XUI хост успешно добавлен!**\n\n🌐 **Хост:** %s\n👤 **Логин:** %s\n🔑 **Пароль:** %s\n%s\n\nХост сохранен в базе данных и готов к использованию.",
		hostData.Host,
		hostData.Login,
		hostData.Password,
		func() string {
			if hostData.SecretKey != "" {
				return fmt.Sprintf("🔐 **Секретный ключ:** %s", hostData.SecretKey)
			}
			return ""
		}())

	return p.sendMessageHTML(client, chatID, successMessage)
}

// handleDefaultMessage обрабатывает сообщения в обычном состоянии
func (p *MessageProcessor) handleDefaultMessage(client *TelegramClient, chatID int, userID int64, username, messageText string, userState *contracts.UserStateInfo) error {
	// Проверяем права пользователя
	canPerform, reason, err := p.userStateService.CanUserPerformAction(userID)
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка проверки прав пользователя: %v", err)
		return p.sendErrorMessage(client, chatID, "Ошибка проверки прав")
	}

	if !canPerform {
		return p.sendMessage(client, chatID, fmt.Sprintf("❌ **Доступ ограничен:** %s", reason))
	}

	// В обычном состоянии просто отправляем информацию о доступных командах
	message := fmt.Sprintf("📋 **Доступные команды:**\n\n/help - Показать справку\n/cancel - Отменить текущую операцию\n\n👤 **Ваше состояние:** %s", userState.State)

	// Добавляем кнопку для администраторов
	if p.adminService.IsGlobalAdmin(userID) {
		message += `\n\n🔧 **Команды администратора:**`
		return p.sendMessageWithInlineAddHost(client, chatID, message)
	}

	return p.sendMessageHTML(client, chatID, message)
}

// sendMessage отправляет обычное сообщение
func (p *MessageProcessor) sendMessage(client *TelegramClient, chatID int, text string) error {
	_, err := client.SendMessage(chatID, text, "")
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка отправки сообщения: %v", err)
	}
	return err
}

// sendMessageHTML отправляет сообщение с HTML разметкой
func (p *MessageProcessor) sendMessageHTML(client *TelegramClient, chatID int, text string) error {
	_, err := client.SendMessageHTML(chatID, text)
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка отправки HTML сообщения: %v", err)
	}
	return err
}

// sendErrorMessage отправляет сообщение об ошибке
func (p *MessageProcessor) sendErrorMessage(client *TelegramClient, chatID int, text string) error {
	errorMessage := fmt.Sprintf("❌ **Ошибка:** %s", text)
	return p.sendMessageHTML(client, chatID, errorMessage)
}

// sendMessageWithInlineAddHost отправляет сообщение с inline-кнопкой для добавления XUI хоста
func (p *MessageProcessor) sendMessageWithInlineAddHost(client *TelegramClient, chatID int, text string) error {
	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{
					Text:         "➕ Добавить XUI хост",
					CallbackData: "addhost",
				},
			},
		},
	}
	_, err := client.SendMessageWithKeyboard(chatID, text, keyboard)
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка отправки сообщения с inline-кнопкой: %v", err)
	}
	return err
}

// sendMessageWithCreateVPNButton отправляет сообщение с кнопкой создания VPN
func (p *MessageProcessor) sendMessageWithCreateVPNButton(client *TelegramClient, chatID int, text string) error {
	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{
					Text:         "🔑 Создать VPN",
					CallbackData: "create_vpn",
				},
			},
		},
	}
	_, err := client.SendMessageWithKeyboard(chatID, text, keyboard)
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка отправки сообщения с кнопкой создания VPN: %v", err)
	}
	return err
}

// handleCreateVPN обрабатывает callback create_vpn
func (p *MessageProcessor) handleCreateVPN(client *TelegramClient, chatID int, userID int64) error {
	servers, err := p.xuiServerService.GetActiveServers()
	if err != nil || len(servers) == 0 {
		return p.sendMessageHTML(client, chatID, "❌ Нет доступных XUI хостов для создания VPN. Обратитесь к администратору.")
	}
	// Случайный выбор
	rnd := time.Now().UnixNano()
	idx := int(rnd) % len(servers)
	server := servers[idx]

	// Создаём клиента XUI
	xui := xui_client.NewClient(server.ServerURL, server.Username, server.Password)
	if err := xui.Login(); err != nil {
		return p.sendMessageHTML(client, chatID, fmt.Sprintf("❌ Ошибка авторизации на XUI: %v", err))
	}

	// Внутренняя функция генерации пароля
	RandString := func(n int) string {
		letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
		b := make([]rune, n)
		for i := range b {
			b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		}
		return string(b)
	}

	vpnLogin := fmt.Sprintf("user%d", userID)
	vpnPass := RandString(12)
	inboundForm := &xui_client.InboundAddForm{
		Remark:   fmt.Sprintf("VPN_%d", userID),
		Protocol: "vmess",
		Port:     "0", // авто
		Settings: fmt.Sprintf(`{"clients":[{"id":"%s","alterId":0}]}`, RandString(32)),
	}
	inboundID, err := xui.AddInbound(inboundForm)
	if err != nil || inboundID == 0 {
		return p.sendMessageHTML(client, chatID, "❌ Не удалось создать VPN на сервере. Попробуйте позже.")
	}
	clientForm := &xui_client.AddClientForm{
		Id:       inboundID,
		Settings: fmt.Sprintf(`{"clients":[{"id":"%s","email":"%s","flow":"","limitIp":0,"totalGB":0,"expiryTime":0,"enable":true,"tgId":0,"subId":"","comment":"","reset":0}]}`, RandString(32), vpnLogin),
	}
	if err := xui.AddClientToInbound(clientForm); err != nil {
		return p.sendMessageHTML(client, chatID, "❌ Не удалось добавить пользователя к VPN. Попробуйте позже.")
	}

	vpnInfo := fmt.Sprintf("✅ VPN создан!\n\n🌐 Сервер: %s\n👤 Логин: <code>%s</code>\n🔑 Пароль: <code>%s</code>\n\nСкопируйте эти данные для подключения.", server.ServerURL, vpnLogin, vpnPass)
	return p.sendMessageHTML(client, chatID, vpnInfo)
}

// handleMonitorCommand обрабатывает команду /monitor
func (p *MessageProcessor) handleMonitorCommand(client *TelegramClient, chatID int, userID int64, username string, userState *contracts.UserStateInfo) error {
	// Проверяем, является ли пользователь глобальным админом
	if !p.adminService.IsGlobalAdmin(userID) {
		return p.sendMessage(client, chatID, "❌ Только глобальные администраторы могут управлять мониторингом хостов.")
	}

	status := p.hostMonitorService.GetMonitoringStatus()
	isRunning := status["is_running"].(bool)
	checkInterval := status["check_interval"].(string)

	message := fmt.Sprintf("🔍 **Мониторинг хостов**\n\n📊 **Статус:** %s\n⏱️ **Интервал проверки:** %s\n\n**Команды управления:**\n/monitor_start - Запустить мониторинг\n/monitor_stop - Остановить мониторинг\n/monitor_status - Подробный статус\n/check_hosts - Проверить все хосты сейчас",
		func() string {
			if isRunning {
				return "🟢 Запущен"
			}
			return "🔴 Остановлен"
		}(),
		checkInterval)

	return p.sendMessageHTML(client, chatID, message)
}

// handleMonitorStartCommand обрабатывает команду /monitor_start
func (p *MessageProcessor) handleMonitorStartCommand(client *TelegramClient, chatID int, userID int64, username string, userState *contracts.UserStateInfo) error {
	// Проверяем, является ли пользователь глобальным админом
	if !p.adminService.IsGlobalAdmin(userID) {
		return p.sendMessage(client, chatID, "❌ Только глобальные администраторы могут запускать мониторинг хостов.")
	}

	if err := p.hostMonitorService.Start(); err != nil {
		log.Printf("[MessageProcessor] Ошибка запуска мониторинга: %v", err)
		return p.sendErrorMessage(client, chatID, fmt.Sprintf("Ошибка запуска мониторинга: %v", err))
	}

	return p.sendMessageHTML(client, chatID, "🟢 **Мониторинг хостов запущен!**\n\nСистема будет автоматически проверять доступность всех активных хостов и уведомлять о проблемах.")
}

// handleMonitorStopCommand обрабатывает команду /monitor_stop
func (p *MessageProcessor) handleMonitorStopCommand(client *TelegramClient, chatID int, userID int64, username string, userState *contracts.UserStateInfo) error {
	// Проверяем, является ли пользователь глобальным админом
	if !p.adminService.IsGlobalAdmin(userID) {
		return p.sendMessage(client, chatID, "❌ Только глобальные администраторы могут останавливать мониторинг хостов.")
	}

	if err := p.hostMonitorService.Stop(); err != nil {
		log.Printf("[MessageProcessor] Ошибка остановки мониторинга: %v", err)
		return p.sendErrorMessage(client, chatID, fmt.Sprintf("Ошибка остановки мониторинга: %v", err))
	}

	return p.sendMessageHTML(client, chatID, "🔴 **Мониторинг хостов остановлен!**\n\nАвтоматические проверки хостов прекращены.")
}

// handleMonitorStatusCommand обрабатывает команду /monitor_status
func (p *MessageProcessor) handleMonitorStatusCommand(client *TelegramClient, chatID int, userID int64, username string, userState *contracts.UserStateInfo) error {
	// Проверяем, является ли пользователь глобальным админом
	if !p.adminService.IsGlobalAdmin(userID) {
		return p.sendMessage(client, chatID, "❌ Только глобальные администраторы могут просматривать статус мониторинга.")
	}

	status := p.hostMonitorService.GetMonitoringStatus()
	isRunning := status["is_running"].(bool)
	checkInterval := status["check_interval"].(string)

	// Получаем статистику хостов
	activeServers, err := p.xuiServerService.GetActiveServers()
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка получения активных серверов: %v", err)
		return p.sendErrorMessage(client, chatID, "Ошибка получения статистики хостов")
	}

	inactiveServers, err := p.xuiServerService.GetInactiveServers()
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка получения неактивных серверов: %v", err)
		return p.sendErrorMessage(client, chatID, "Ошибка получения статистики хостов")
	}

	message := fmt.Sprintf("📊 **Статус мониторинга хостов**\n\n🔄 **Мониторинг:** %s\n⏱️ **Интервал проверки:** %s\n\n📈 **Статистика хостов:**\n🟢 Активных: %d\n🔴 Неактивных: %d\n📊 Всего: %d\n\n%s",
		func() string {
			if isRunning {
				return "🟢 Запущен"
			}
			return "🔴 Остановлен"
		}(),
		checkInterval,
		len(activeServers),
		len(inactiveServers),
		len(activeServers)+len(inactiveServers),
		func() string {
			if len(inactiveServers) > 0 {
				return "⚠️ **Неактивные хосты:**\n" + func() string {
					var hosts string
					for i, server := range inactiveServers {
						if i >= 5 { // Показываем только первые 5
							hosts += fmt.Sprintf("... и еще %d хостов\n", len(inactiveServers)-5)
							break
						}
						hosts += fmt.Sprintf("• %s (%s)\n", server.ServerName, server.ServerURL)
					}
					return hosts
				}()
			}
			return ""
		}())

	return p.sendMessageHTML(client, chatID, message)
}

// handleCheckHostsCommand обрабатывает команду /check_hosts
func (p *MessageProcessor) handleCheckHostsCommand(client *TelegramClient, chatID int, userID int64, username string, userState *contracts.UserStateInfo) error {
	// Проверяем, является ли пользователь глобальным админом
	if !p.adminService.IsGlobalAdmin(userID) {
		return p.sendMessage(client, chatID, "❌ Только глобальные администраторы могут выполнять проверку хостов.")
	}

	// Отправляем сообщение о начале проверки
	startMessage := "🔍 **Начинаем проверку всех активных хостов...**\n\nЭто может занять некоторое время."
	if err := p.sendMessageHTML(client, chatID, startMessage); err != nil {
		return err
	}

	// Получаем все активные серверы
	servers, err := p.xuiServerService.GetActiveServers()
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка получения активных серверов: %v", err)
		return p.sendErrorMessage(client, chatID, "Ошибка получения списка хостов")
	}

	if len(servers) == 0 {
		return p.sendMessageHTML(client, chatID, "ℹ️ **Нет активных хостов для проверки**")
	}

	// Проверяем каждый хост
	var results []string
	var inactiveCount int

	for _, server := range servers {
		status, err := p.hostMonitorService.CheckHostNow(server.ID)
		if err != nil {
			log.Printf("[MessageProcessor] Ошибка проверки хоста %d: %v", server.ID, err)
			results = append(results, fmt.Sprintf("❌ %s - Ошибка проверки: %v", server.ServerName, err))
			inactiveCount++
			continue
		}

		if status.IsActive {
			results = append(results, fmt.Sprintf("🟢 %s - Активен", server.ServerName))
		} else {
			results = append(results, fmt.Sprintf("🔴 %s - Неактивен (%s)", server.ServerName, status.Error))
			inactiveCount++
		}
	}

	// Формируем итоговое сообщение
	summary := fmt.Sprintf("📊 **Результаты проверки хостов**\n\n✅ **Проверено хостов:** %d\n🟢 **Активных:** %d\n🔴 **Неактивных:** %d\n\n**Детали:**", len(servers), len(servers)-inactiveCount, inactiveCount)

	for _, result := range results {
		summary += "\n" + result
	}

	if inactiveCount > 0 {
		summary += "\n\n⚠️ **Неактивные хосты автоматически отключены и не будут использоваться для создания VPN.**"
	}

	return p.sendMessageHTML(client, chatID, summary)
}

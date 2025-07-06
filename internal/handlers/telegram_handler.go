package handlers

import (
	"fmt"
	"log"
	"strings"

	"TelegramXUI/internal/config"
	"TelegramXUI/internal/services"
	"TelegramXUI/internal/telegram"
)

// TelegramHandler обрабатывает сообщения и callback query от Telegram
type TelegramHandler struct {
	vpnService  *services.VPNService
	userService *services.UserService
	telegramBot *telegram.TelegramBot
	webAppURL   string
	vpnConfig   *config.VPNConfig
}

// NewTelegramHandler создает новый обработчик Telegram
func NewTelegramHandler(vpnService *services.VPNService, userService *services.UserService, telegramBot *telegram.TelegramBot, webAppURL string, vpnConfig *config.VPNConfig) *TelegramHandler {
	return &TelegramHandler{
		vpnService:  vpnService,
		userService: userService,
		telegramBot: telegramBot,
		webAppURL:   webAppURL,
		vpnConfig:   vpnConfig,
	}
}

// HandleMessage обрабатывает входящие сообщения
func (h *TelegramHandler) HandleMessage(client *telegram.TelegramClient, update telegram.Update) error {
	// Проверяем, что это callback query
	if update.CallbackQuery != nil && update.CallbackQuery.ID != "" {
		return h.handleCallbackQuery(client, update)
	}

	// Проверяем, что есть сообщение
	if update.Message == nil {
		return nil
	}

	if update.Message.Text == "" {
		return nil
	}

	chatID := update.Message.Chat.ID
	text := strings.ToLower(strings.TrimSpace(update.Message.Text))
	userName := update.Message.From.FirstName

	log.Printf("[MessageHandler] Обработка сообщения от %s (ID: %d): %s", userName, chatID, update.Message.Text)

	// Проверяем/создаем пользователя в базе данных
	if h.userService != nil {
		user, err := h.userService.EnsureUserExists(update.Message.From)
		if err != nil {
			log.Printf("[MessageHandler] Ошибка работы с пользователем: %v", err)
		} else {
			if user != nil {
				log.Printf("[MessageHandler] Пользователь %s (ID: %d) обработан в БД", userName, chatID)
			}
		}
	}

	// Обработка команд
	switch {
	case text == "/start" || text == "start":
		return h.handleStartCommand(client, chatID, userName)

	case text == "/webapp" || text == "webapp":
		return h.handleWebAppCommand(client, chatID)

	case text == "/help" || text == "help":
		return h.handleHelpCommand(client, chatID)

	case text == "/status" || text == "status":
		return h.handleStatusCommand(client, chatID)

	case text == "/users" || text == "users":
		return h.handleUsersCommand(client, chatID)

	default:
		return h.handleUnknownCommand(client, chatID, update.Message.Text)
	}
}

// handleStartCommand обрабатывает команду /start
func (h *TelegramHandler) handleStartCommand(client *telegram.TelegramClient, chatID int, userName string) error {
	log.Printf("[MessageHandler] Выполняется команда /start для пользователя %s (ID: %d)", userName, chatID)

	message := fmt.Sprintf(`Привет, %s! 👋

Добро пожаловать в VPN Manager Bot!

Здесь вы можете:
• Создавать VPN подключения
• Управлять своими подключениями
• Просматривать статистику

Нажмите кнопку ниже, чтобы создать VPN подключение:`, userName)

	_, err := client.SendMessageWithKeyboard(chatID, message, h.createWelcomeKeyboard())
	if err != nil {
		log.Printf("[MessageHandler] Ошибка отправки приветственного сообщения: %v", err)
		// Fallback на обычное сообщение
		fallbackMessage := fmt.Sprintf("Привет, %s! 👋\n\nЯ бот для управления VPN через x-ui.\n\nДоступные команды:\n/status - Статус x-ui\n/help - Помощь\n\n💡 Используйте /start для получения меню с кнопками!", userName)
		log.Printf("[MessageHandler] Отправка fallback сообщения для пользователя %s (ID: %d)", userName, chatID)
		return h.telegramBot.SendMessageHTML(chatID, fallbackMessage)
	}

	log.Printf("[MessageHandler] Приветственное сообщение с inline кнопками отправлено успешно для пользователя %s (ID: %d)", userName, chatID)
	return nil
}

// handleWebAppCommand обрабатывает команду /webapp
func (h *TelegramHandler) handleWebAppCommand(client *telegram.TelegramClient, chatID int) error {
	log.Printf("[MessageHandler] Выполняется команда /webapp для пользователя (ID: %d)", chatID)
	message := "🚀 Нажмите кнопку ниже, чтобы открыть панель управления VPN:"
	_, err := client.SendMessageWithWebAppButton(chatID, message, "📱 Открыть панель", h.webAppURL)
	if err != nil {
		log.Printf("[MessageHandler] Ошибка отправки WebApp кнопки: %v", err)
		return h.telegramBot.SendMessage(chatID, "Ошибка отправки кнопки. Попробуйте позже.")
	}
	log.Printf("[MessageHandler] WebApp кнопка отправлена успешно для пользователя (ID: %d)", chatID)
	return nil
}

// handleHelpCommand обрабатывает команду /help
func (h *TelegramHandler) handleHelpCommand(client *telegram.TelegramClient, chatID int) error {
	log.Printf("[MessageHandler] Выполняется команда /help для пользователя (ID: %d)", chatID)
	message := `<b>Доступные команды:</b>

/start - Начать работу с ботом (откроет меню с кнопками)
/webapp - Открыть панель управления VPN
/status - Проверить статус x-ui
/users - Статистика пользователей
/help - Показать эту справку

<b>💡 Совет:</b> Используйте команду /start для получения удобного меню с кнопками для создания VPN!

<i>Разработка в процессе...</i>`
	log.Printf("[MessageHandler] Отправка справки для пользователя (ID: %d)", chatID)
	return h.telegramBot.SendMessageHTML(chatID, message)
}

// handleStatusCommand обрабатывает команду /status
func (h *TelegramHandler) handleStatusCommand(client *telegram.TelegramClient, chatID int) error {
	log.Printf("[MessageHandler] Выполняется команда /status для пользователя (ID: %d)", chatID)
	if h.vpnService != nil {
		if err := h.vpnService.CheckStatus(); err != nil {
			message := fmt.Sprintf("❌ <b>Ошибка подключения к x-ui:</b>\n%s", err.Error())
			log.Printf("[MessageHandler] Ошибка x-ui для пользователя (ID: %d): %v", chatID, err)
			return h.telegramBot.SendMessageHTML(chatID, message)
		} else {
			message := "✅ <b>Статус x-ui:</b>\nПодключение успешно установлено"
			log.Printf("[MessageHandler] Статус x-ui успешен для пользователя (ID: %d)", chatID)
			return h.telegramBot.SendMessageHTML(chatID, message)
		}
	} else {
		message := "❌ <b>Статус x-ui:</b>\nКлиент не инициализирован"
		log.Printf("[MessageHandler] x-ui клиент не инициализирован для пользователя (ID: %d)", chatID)
		return h.telegramBot.SendMessageHTML(chatID, message)
	}
}

// handleUsersCommand обрабатывает команду /users
func (h *TelegramHandler) handleUsersCommand(client *telegram.TelegramClient, chatID int) error {
	log.Printf("[MessageHandler] Выполняется команда /users для пользователя (ID: %d)", chatID)
	if h.userService != nil {
		count, err := h.userService.GetUsersCount()
		if err != nil {
			log.Printf("[MessageHandler] Ошибка получения количества пользователей: %v", err)
			return h.telegramBot.SendMessage(chatID, "Ошибка получения статистики пользователей.")
		}

		message := fmt.Sprintf("📊 <b>Статистика пользователей:</b>\n\nВсего пользователей: %d", count)
		log.Printf("[MessageHandler] Статистика пользователей отправлена для пользователя (ID: %d): %d пользователей", chatID, count)
		return h.telegramBot.SendMessageHTML(chatID, message)
	} else {
		log.Printf("[MessageHandler] Сервис пользователей не инициализирован для пользователя (ID: %d)", chatID)
		return h.telegramBot.SendMessage(chatID, "Сервис пользователей не инициализирован.")
	}
}

// handleUnknownCommand обрабатывает неизвестные команды
func (h *TelegramHandler) handleUnknownCommand(client *telegram.TelegramClient, chatID int, text string) error {
	log.Printf("[MessageHandler] Неизвестная команда от пользователя (ID: %d): %s", chatID, text)
	message := fmt.Sprintf("Неизвестная команда: %s\n\nИспользуйте /help для получения справки", text)
	return h.telegramBot.SendMessage(chatID, message)
}

// handleCallbackQuery обрабатывает callback query от inline кнопок
func (h *TelegramHandler) handleCallbackQuery(client *telegram.TelegramClient, update telegram.Update) error {
	callbackQuery := update.CallbackQuery
	if callbackQuery == nil {
		return fmt.Errorf("callback query is nil")
	}

	chatID := callbackQuery.From.ID
	userName := callbackQuery.From.FirstName
	if userName == "" {
		userName = callbackQuery.From.Username
	}
	if userName == "" {
		userName = fmt.Sprintf("User%d", callbackQuery.From.ID)
	}

	log.Printf("[CallbackHandler] Получен callback query от пользователя %s (ID: %d): %s", userName, chatID, callbackQuery.Data)

	// Отвечаем на callback query
	if _, err := client.AnswerCallbackQuery(callbackQuery.ID, ""); err != nil {
		log.Printf("[CallbackHandler] Ошибка ответа на callback query: %v", err)
	}

	// Обработка различных типов callback query
	switch callbackQuery.Data {
	case CallbackCreateVPN:
		return h.handleCreateVPNCallback(client, chatID, userName)

	default:
		log.Printf("[CallbackHandler] Неизвестный callback query от пользователя %s (ID: %d): %s", userName, chatID, callbackQuery.Data)
		return h.telegramBot.SendMessage(chatID, "Неизвестное действие. Попробуйте еще раз.")
	}
}

// handleCreateVPNCallback обрабатывает callback для создания VPN
func (h *TelegramHandler) handleCreateVPNCallback(client *telegram.TelegramClient, chatID int, userName string) error {
	log.Printf("[CallbackHandler] Создание VPN для пользователя %s (ID: %d)", userName, chatID)

	// Получаем информацию о пользователе
	user, err := h.userService.GetUserByTelegramID(int64(chatID))
	if err != nil {
		log.Printf("[CallbackHandler] Ошибка получения пользователя %d: %v", chatID, err)
		return h.telegramBot.SendMessageHTML(chatID, "❌ <b>Ошибка получения данных пользователя</b>")
	}

	// Создаем VPN для пользователя
	vpnConnection, err := h.vpnService.CreateVPNForUser(
		int64(chatID),
		user.Username,
		user.FirstName,
		user.LastName,
		1, // TODO: Получать ServerID из конфигурации или выбора
		h.vpnConfig,
	)
	if err != nil {
		log.Printf("[CallbackHandler] Ошибка создания VPN для пользователя %s (ID: %d): %v", userName, chatID, err)
		errorMessage := fmt.Sprintf("❌ <b>Ошибка создания VPN:</b>\n%s", err.Error())
		return h.telegramBot.SendMessageHTML(chatID, errorMessage)
	}

	// Отправляем информацию о созданном VPN
	return h.sendVPNConnectionInfo(client, chatID, userName, vpnConnection)
}

// createWelcomeKeyboard создает клавиатуру для приветственного сообщения
func (h *TelegramHandler) createWelcomeKeyboard() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{
					Text:         "🔐 Создать VPN",
					CallbackData: CallbackCreateVPN,
				},
			},
		},
	}
}

// createVPNSuccessKeyboard создает клавиатуру для сообщения об успешном создании VPN
func (h *TelegramHandler) createVPNSuccessKeyboard() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{
					Text:         "🔐 Создать еще VPN",
					CallbackData: CallbackCreateVPN,
				},
			},
		},
	}
}

// sendVPNConnectionInfo отправляет информацию о созданном VPN подключении
func (h *TelegramHandler) sendVPNConnectionInfo(client *telegram.TelegramClient, chatID int, userName string, vpnConnection *services.VPNConnection) error {
	// Формируем сообщение с данными подключения и кнопкой для создания еще одного VPN
	message := h.createVPNSuccessMessage(userName, chatID, vpnConnection)

	log.Printf("[VPN] VPN создан успешно для пользователя %s (ID: %d)", userName, chatID)

	// Отправляем основное сообщение с кнопкой
	if _, err := client.SendMessageWithKeyboard(chatID, message, h.createVPNSuccessKeyboard()); err != nil {
		log.Printf("[VPN] Ошибка отправки основного сообщения: %v", err)
		return err
	}

	// Отправляем данные для подключения в отдельном сообщении
	connectionMessage := h.createVPNConnectionMessage(vpnConnection)

	log.Printf("[VPN] Отправка данных подключения для пользователя %s (ID: %d)", userName, chatID)
	_, err := client.SendMessageHTML(chatID, connectionMessage)
	return err
}

// createVPNSuccessMessage создает сообщение об успешном создании VPN
func (h *TelegramHandler) createVPNSuccessMessage(userName string, chatID int, vpnConnection *services.VPNConnection) string {
	return fmt.Sprintf(`✅ <b>VPN подключение создано!</b>

🔒 <b>VPN #%d</b>
👤 <b>Пользователь:</b> %s
🆔 <b>ID:</b> %d
📧 <b>Email:</b> %s
🔌 <b>Порт:</b> %d
📅 <b>Создано:</b> %s

<i>Данные для подключения отправлены в следующем сообщении.</i>`,
		vpnConnection.ID, userName, chatID, vpnConnection.Email, vpnConnection.Port, vpnConnection.CreatedAt.Format("02.01.2006 15:04:05"))
}

// createVPNConnectionMessage создает сообщение с данными для подключения
func (h *TelegramHandler) createVPNConnectionMessage(vpnConnection *services.VPNConnection) string {
	return fmt.Sprintf(`🔗 <b>VLESS ссылка для подключения:</b>

%s

📱 <b>Для подключения:</b>
1. Скопируйте VLESS ссылку выше
2. Откройте приложение V2rayNG или аналогичное
3. Нажмите «+» и выберите «Импорт из буфера обмена»
4. Вставьте ссылку и нажмите «Сохранить»

⚠️ <b>Важно:</b> Храните эту ссылку в безопасном месте!`,
		vpnConnection.VlessLink)
}

// Константы для callback_data
const (
	CallbackCreateVPN = "create_vpn"
)

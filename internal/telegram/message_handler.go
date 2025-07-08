package telegram

import (
	"TelegramXUI/internal/config"
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
	userService            *UserService
	vpnConnectionService   *services.VPNConnectionService
	config                 *config.Config
	transactionService     *services.TransactionService
}

// NewMessageProcessor создает новый обработчик сообщений
func NewMessageProcessor(
	userStateService contracts.UserStateService,
	extensibleStateService contracts.ExtensibleStateService,
	xuiHostAddService contracts.XUIHostAddService,
	adminService contracts.AdminService,
	xuiServerService *services.XUIServerService,
	hostMonitorService *services.HostMonitorService,
	userService *UserService,
	vpnConnectionService *services.VPNConnectionService,
	config *config.Config,
	transactionService *services.TransactionService,
) *MessageProcessor {
	return &MessageProcessor{
		userStateService:       userStateService,
		extensibleStateService: extensibleStateService,
		xuiHostAddService:      xuiHostAddService,
		adminService:           adminService,
		xuiServerService:       xuiServerService,
		hostMonitorService:     hostMonitorService,
		userService:            userService,
		vpnConnectionService:   vpnConnectionService,
		config:                 config,
		transactionService:     transactionService,
	}
}

// ProcessMessage обрабатывает входящее сообщение
func (p *MessageProcessor) ProcessMessage(client *TelegramClient, update Update) error {
	log.Printf("[RAW UPDATE] %+v", update)

	// СНАЧАЛА обработка оплаты!
	if update.PreCheckoutQuery != nil {
		log.Printf("[DEBUG] pre_checkout_query: id=%s, user_id=%d, currency=%s, total_amount=%d, payload=%s", update.PreCheckoutQuery.ID, update.PreCheckoutQuery.From.ID, update.PreCheckoutQuery.Currency, update.PreCheckoutQuery.TotalAmount, update.PreCheckoutQuery.InvoicePayload)
		// Подтверждаем pre_checkout_query и уведомляем пользователя
		err := client.AnswerPreCheckoutQuery(update.PreCheckoutQuery.ID, true, "")
		if err != nil {
			log.Printf("[MessageProcessor] Ошибка подтверждения pre_checkout_query: %v", err)
		} else {
			log.Printf("[MessageProcessor] pre_checkout_query подтверждён для user_id=%d", update.PreCheckoutQuery.From.ID)
		}
		chatID := int(update.PreCheckoutQuery.From.ID)
		_ = p.sendMessageHTML(client, chatID, "💸 Запрос на оплату получен, ожидайте подтверждения!")
		return nil
	}

	if update.Message != nil && update.Message.SuccessfulPayment != nil {
		log.Printf("[DEBUG] Получен SuccessfulPayment: %+v", update)
		if update.Message.From.ID == 0 {
			log.Printf("[ERROR] update.Message.SuccessfulPayment, но From.ID == 0: %+v", update)
			return p.sendErrorMessage(client, 0, "Ошибка: не удалось определить пользователя для оплаты")
		}
		userID := int64(update.Message.From.ID)
		chatID := update.Message.Chat.ID
		log.Printf("[DEBUG] SuccessfulPayment userID=%d, chatID=%d", userID, chatID)

		// Сообщаем пользователю, что платёж принят и идёт создание VPN
		errMsg := p.sendMessageHTML(client, chatID, "⭐️ Платёж успешно принят! Создаём VPN...")
		if errMsg != nil {
			log.Printf("[MessageProcessor] Ошибка отправки сообщения о принятии платежа: %v", errMsg)
		}

		// Пробуем создать VPN
		errVPN := p.createVPNAndSendInfo(client, chatID, userID)
		sp := update.Message.SuccessfulPayment
		if errVPN != nil {
			// Если не удалось — делаем возврат
			log.Printf("[ERROR] Ошибка создания VPN: %v", errVPN)
			refundErr := client.RefundStarPayment(userID, sp.TelegramPaymentChargeID, sp.TotalAmount, "Не удалось создать VPN, возврат средств")
			if refundErr != nil {
				log.Printf("[ERROR] Ошибка возврата средств: %v", refundErr)
				p.sendMessageHTML(client, chatID, "❌ Не удалось создать VPN и вернуть средства. Обратитесь к администратору.")
			} else {
				p.sendMessageHTML(client, chatID, "❌ Не удалось создать VPN. Ваши средства возвращены.")
			}
			return nil
		}

		// Если VPN создан — записываем транзакцию
		trx := &services.Transaction{
			TelegramPaymentChargeID: sp.TelegramPaymentChargeID,
			TelegramUserID:          userID,
			Amount:                  sp.TotalAmount,
			InvoicePayload:          sp.InvoicePayload,
			Status:                  "success",
			Type:                    "payment",
			Reason:                  "Оплата через Telegram Stars",
		}
		errTrx := p.transactionService.AddTransaction(trx)
		if errTrx != nil {
			log.Printf("[ERROR] Ошибка записи транзакции: %v", errTrx)
		}

		return nil
	}

	// Обрабатываем только текстовые сообщения и callback queries
	if update.Message == nil && update.CallbackQuery == nil {
		return nil
	}

	var userID int64
	var username string
	var messageText string
	var chatID int
	var firstName string
	var lastName string
	var isBot bool

	// Извлекаем данные пользователя
	if update.Message != nil {
		userID = int64(update.Message.From.ID)
		username = update.Message.From.Username
		firstName = update.Message.From.FirstName
		lastName = update.Message.From.LastName
		isBot = update.Message.From.IsBot
		messageText = update.Message.Text
		chatID = update.Message.Chat.ID
	} else if update.CallbackQuery != nil {
		userID = int64(update.CallbackQuery.From.ID)
		username = update.CallbackQuery.From.Username
		firstName = update.CallbackQuery.From.FirstName
		lastName = update.CallbackQuery.From.LastName
		isBot = update.CallbackQuery.From.IsBot
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
		} else if messageText == "check_hosts" {
			// Получаем состояние пользователя для callback-запроса
			userState, err := p.userStateService.GetUserState(userID)
			if err != nil {
				log.Printf("[MessageProcessor] Ошибка получения состояния пользователя %d: %v", userID, err)
				return p.sendErrorMessage(client, chatID, "Ошибка получения состояния пользователя")
			}

			// Отвечаем на callback-запрос
			if _, err := client.AnswerCallbackQuery(update.CallbackQuery.ID, ""); err != nil {
				log.Printf("[MessageProcessor] Ошибка ответа на callback check_hosts: %v", err)
			}

			return p.handleCheckHostsCommand(client, chatID, userID, username, userState)
		} else if strings.HasPrefix(messageText, "vpn_") {
			// Отвечаем на callback-запрос
			if _, err := client.AnswerCallbackQuery(update.CallbackQuery.ID, ""); err != nil {
				log.Printf("[MessageProcessor] Ошибка ответа на callback vpn: %v", err)
			}

			return p.handleVPNCallback(client, chatID, userID, messageText)
		} else if strings.HasPrefix(messageText, "refund_") {
			if _, err := client.AnswerCallbackQuery(update.CallbackQuery.ID, ""); err != nil {
				log.Printf("[MessageProcessor] Ошибка ответа на callback refund: %v", err)
			}
			return p.handleRefundCallback(client, chatID, userID, messageText)
		}
	}

	log.Printf("[MessageProcessor] Обработка сообщения от пользователя %d (%s): %s", userID, username, messageText)

	// Проверяем состояние пользователя
	userState, err := p.userStateService.GetUserState(userID)
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка получения состояния пользователя %d: %v", userID, err)
		return p.sendErrorMessage(client, chatID, "Ошибка получения состояния пользователя")
	}

	// Если пользователь не найден, автоматически создаем его с активным состоянием
	if userState == nil {
		log.Printf("[MessageProcessor] Пользователь %d не найден, автоматически создаем активное состояние", userID)

		// Создаем пользователя с активным состоянием
		err = p.createUserWithActiveState(userID, username, firstName, lastName, isBot)
		if err != nil {
			log.Printf("[MessageProcessor] Ошибка создания пользователя %d: %v", userID, err)
			return p.sendErrorMessage(client, chatID, "Ошибка создания пользователя")
		}

		// Получаем созданное состояние пользователя
		userState, err = p.userStateService.GetUserState(userID)
		if err != nil {
			log.Printf("[MessageProcessor] Ошибка получения созданного состояния пользователя %d: %v", userID, err)
			return p.sendErrorMessage(client, chatID, "Ошибка получения состояния пользователя")
		}

		log.Printf("[MessageProcessor] Пользователь %d успешно создан с активным состоянием", userID)
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

	return nil
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
	case "/vpn":
		return p.handleVPNCommand(client, chatID, userID, username, userState)
	case "/transactions":
		return p.handleTransactionsCommand(client, chatID, userID)
	default:
		return p.sendMessage(client, chatID, fmt.Sprintf("Неизвестная команда: %s\nИспользуйте /help для получения справки", command))
	}
}

// handleStartCommand обрабатывает команду /start
func (p *MessageProcessor) handleStartCommand(client *TelegramClient, chatID int, userID int64, username string, userState *contracts.UserStateInfo) error {
	message := fmt.Sprintf(`🤖 <b>Добро пожаловать в TelegramXUI!</b>

👤 <b>Пользователь:</b> %s
📊 <b>Состояние:</b> %s

✅ <b>Регистрация:</b> Автоматически завершена
🎯 <b>Доступ:</b> Полный доступ к функциям

Используйте /help для получения справки.`, username, userState.State)
	if p.adminService.IsGlobalAdmin(userID) {
		message += `\n\n🔧 <b>Администратор:</b> Доступны расширенные функции`
		return p.sendMessageWithAdminButtons(client, chatID, message)
	}
	return p.sendMessageWithCreateVPNButton(client, chatID, message)
}

// handleHelpCommand обрабатывает команду /help
func (p *MessageProcessor) handleHelpCommand(client *TelegramClient, chatID int, userID int64, username string, userState *contracts.UserStateInfo) error {
	message := `📚 <b>Справка по командам:</b>

/start - Начать работу с ботом
/help - Показать эту справку
/cancel - Отменить текущую операцию
/vpn - Управление VPN подключениями

💡 <b>Совет:</b> Нажмите кнопку "Создать VPN" для быстрого доступа к VPN`

	if p.adminService.IsGlobalAdmin(userID) {
		message += `\n\n🔧 <b>Команды администратора:</b>
/addhost - Добавить XUI хост
/monitor - Управление мониторингом хостов
/monitor_start - Запустить мониторинг
/monitor_stop - Остановить мониторинг
/monitor_status - Статус мониторинга
/check_hosts - Проверить все хосты сейчас

💡 <b>Быстрые действия:</b> Используйте кнопки ниже для быстрого доступа`
		return p.sendMessageWithAdminButtons(client, chatID, message)
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
		errorMessage := fmt.Sprintf("❌ <b>Ошибка:</b> %v\n\nПопробуйте еще раз или отправьте /cancel для отмены.", err)
		return p.sendMessageHTML(client, chatID, errorMessage)
	}

	// Успешно добавлен хост
	successMessage := fmt.Sprintf("✅ <b>XUI хост успешно добавлен!</b>\n\n🌐 <b>Хост:</b> %s\n👤 <b>Логин:</b> %s\n🔑 <b>Пароль:</b> %s\n%s\n\nХост сохранен в базе данных и готов к использованию.",
		hostData.Host,
		hostData.Login,
		hostData.Password,
		func() string {
			if hostData.SecretKey != "" {
				return fmt.Sprintf("🔐 <b>Секретный ключ:</b> %s", hostData.SecretKey)
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
		return p.sendMessage(client, chatID, fmt.Sprintf("❌ <b>Доступ ограничен:</b> %s", reason))
	}

	// В обычном состоянии просто отправляем информацию о доступных командах
	message := fmt.Sprintf("📋 <b>Доступные команды:</b>\n\n/help - Показать справку\n/cancel - Отменить текущую операцию\n\n👤 <b>Ваше состояние:</b> %s", userState.State)

	// Добавляем кнопку для администраторов
	if p.adminService.IsGlobalAdmin(userID) {
		message += `\n\n🔧 <b>Команды администратора:</b>`
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
	errorMessage := fmt.Sprintf("❌ <b>Ошибка:</b> %s", text)
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

// sendMessageWithAdminButtons отправляет сообщение с кнопками для администраторов
func (p *MessageProcessor) sendMessageWithAdminButtons(client *TelegramClient, chatID int, text string) error {
	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{
					Text:         "🔑 Создать VPN",
					CallbackData: "create_vpn",
				},
			},
			{
				{
					Text:         "➕ Добавить XUI хост",
					CallbackData: "addhost",
				},
				{
					Text:         "🔍 Проверить хосты",
					CallbackData: "check_hosts",
				},
			},
		},
	}
	_, err := client.SendMessageWithKeyboard(chatID, text, keyboard)
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка отправки сообщения с кнопками администратора: %v", err)
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
	title := "Создание VPN-подключения"
	description := "Оплата за создание VPN через Telegram Stars"
	payload := fmt.Sprintf("vpn_create_%d", userID)
	providerToken := "" // Для Stars provider_token пустой
	currency := "XTR"   // Валюта Stars
	prices := []LabeledPrice{{Label: "VPN", Amount: 1}}
	isTest := false
	if p.adminService.IsGlobalAdmin(userID) {
		isTest = false // Для админа — тестовый инвойс, чтобы не списывались звёзды
	}
	if err := client.SendInvoice(chatID, title, description, payload, providerToken, currency, prices, isTest); err != nil {
		return p.sendErrorMessage(client, chatID, "Ошибка отправки инвойса на оплату через Stars")
	}
	return nil // Ждём подтверждения оплаты
}

// createVPNAndSendInfo — создание VPN и отправка пользователю
func (p *MessageProcessor) createVPNAndSendInfo(client *TelegramClient, chatID int, userID int64) error {
	user, err := p.userService.GetUserByTelegramID(userID)
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка получения пользователя %d: %v", userID, err)
		return p.sendErrorMessage(client, chatID, "Ошибка получения данных пользователя")
	}
	servers, err := p.xuiServerService.GetActiveServers()
	if err != nil || len(servers) == 0 {
		return p.sendMessageHTML(client, chatID, "❌ Нет доступных XUI хостов для создания VPN. Обратитесь к администратору.")
	}
	rnd := time.Now().UnixNano()
	idx := int(rnd) % len(servers)
	server := servers[idx]
	xui := xui_client.NewClient(server.ServerURL, server.Username, server.Password)
	vpnService := services.NewVPNService(xui, p.vpnConnectionService)
	vpnConnection, err := vpnService.CreateVPNForUser(
		userID,
		user.Username,
		user.FirstName,
		user.LastName,
		server.ID,
		&p.config.VPN,
	)
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка создания VPN для пользователя %d: %v", userID, err)
		return p.sendErrorMessage(client, chatID, fmt.Sprintf("Ошибка создания VPN: %v", err))
	}
	message := fmt.Sprintf("✅ <b>VPN успешно создан и сохранен!</b>\n\n")
	message += fmt.Sprintf("🔒 <b>VPN подключение #%d</b>\n", vpnConnection.ID)
	message += fmt.Sprintf("🌐 <b>Сервер:</b> %s\n", server.ServerName)
	message += fmt.Sprintf("🔌 <b>Порт:</b> %d\n", vpnConnection.Port)
	message += fmt.Sprintf("📧 <b>Email:</b> %s\n", vpnConnection.Email)
	message += fmt.Sprintf("📅 <b>Создано:</b> %s\n\n", vpnConnection.CreatedAt.Format("02.01.2006 15:04:05"))
	message += "🔗 <b>VLESS ссылка для подключения:</b>\n"
	message += fmt.Sprintf("<code>%s</code>\n\n", vpnConnection.VlessLink)
	message += "📱 <b>Для подключения:</b>\n"
	message += "1. Скопируйте VLESS ссылку выше\n"
	message += "2. Откройте приложение V2rayNG или аналогичное\n"
	message += "3. Нажмите «+» и выберите «Импорт из буфера обмена»\n"
	message += "4. Вставьте ссылку и нажмите «Сохранить»\n\n"
	message += "💡 <b>Управление VPN:</b> Используйте команду /vpn для просмотра всех ваших подключений"
	return p.sendMessageHTML(client, chatID, message)
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

	message := fmt.Sprintf("🔍 <b>Мониторинг хостов</b>\n\n📊 <b>Статус:</b> %s\n⏱️ <b>Интервал проверки:</b> %s\n\n<b>Команды управления:</b>\n/monitor_start - Запустить мониторинг\n/monitor_stop - Остановить мониторинг\n/monitor_status - Подробный статус\n/check_hosts - Проверить все хосты сейчас\n\n💡 <b>Быстрые действия:</b> Используйте кнопки ниже",
		func() string {
			if isRunning {
				return "🟢 Запущен"
			}
			return "🔴 Остановлен"
		}(),
		checkInterval)

	return p.sendMessageWithAdminButtons(client, chatID, message)
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

	return p.sendMessageHTML(client, chatID, "🟢 <b>Мониторинг хостов запущен!</b>\n\nСистема будет автоматически проверять доступность всех активных хостов и уведомлять о проблемах.")
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

	return p.sendMessageHTML(client, chatID, "🔴 <b>Мониторинг хостов остановлен!</b>\n\nАвтоматические проверки хостов прекращены.")
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

	message := fmt.Sprintf("📊 <b>Статус мониторинга хостов</b>\n\n🔄 <b>Мониторинг:</b> %s\n⏱️ <b>Интервал проверки:</b> %s\n\n📈 <b>Статистика хостов:</b>\n🟢 Активных: %d\n🔴 Неактивных: %d\n📊 Всего: %d\n\n%s",
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
				return "⚠️ <b>Неактивные хосты:</b>\n" + func() string {
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

	message += "\n\n💡 <b>Быстрые действия:</b> Используйте кнопки ниже для управления"

	return p.sendMessageWithAdminButtons(client, chatID, message)
}

// handleCheckHostsCommand обрабатывает команду /check_hosts
func (p *MessageProcessor) handleCheckHostsCommand(client *TelegramClient, chatID int, userID int64, username string, userState *contracts.UserStateInfo) error {
	// Проверяем, является ли пользователь глобальным админом
	if !p.adminService.IsGlobalAdmin(userID) {
		return p.sendMessage(client, chatID, "❌ Только глобальные администраторы могут выполнять проверку хостов.")
	}

	// Отправляем сообщение о начале проверки
	startMessage := "🔍 <b>Начинаем принудительную проверку всех активных хостов...</b>\n\n⏱️ Это может занять некоторое время."
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
		return p.sendMessageHTML(client, chatID, "ℹ️ <b>Нет активных хостов для проверки</b>\n\nДобавьте хосты с помощью команды /addhost или кнопки «➕ Добавить XUI хост»")
	}

	// Проверяем каждый хост
	var results []string
	var inactiveCount int
	var activeCount int

	log.Printf("[MessageProcessor] Начинаем проверку %d хостов", len(servers))

	for i, server := range servers {
		log.Printf("[MessageProcessor] Проверяем хост %d/%d: %s (%s)", i+1, len(servers), server.ServerName, server.ServerURL)

		status, err := p.hostMonitorService.CheckHostNow(server.ID)
		if err != nil {
			log.Printf("[MessageProcessor] Ошибка проверки хоста %d: %v", server.ID, err)
			results = append(results, fmt.Sprintf("❌ <b>%s</b> - Ошибка проверки: %v", server.ServerName, err))
			inactiveCount++
			continue
		}

		if status.IsActive {
			results = append(results, fmt.Sprintf("🟢 <b>%s</b> - Активен (%s)", server.ServerName, server.ServerURL))
			activeCount++
		} else {
			results = append(results, fmt.Sprintf("🔴 <b>%s</b> - Неактивен (%s)\n   Причина: %s", server.ServerName, server.ServerURL, status.Error))
			inactiveCount++
		}
	}

	// Формируем итоговое сообщение
	summary := fmt.Sprintf("📊 <b>Результаты принудительной проверки хостов</b>\n\n⏰ <b>Время проверки:</b> %s\n✅ <b>Проверено хостов:</b> %d\n🟢 <b>Активных:</b> %d\n🔴 <b>Неактивных:</b> %d\n\n<b>Детали проверки:</b>",
		time.Now().Format("15:04:05"), len(servers), activeCount, inactiveCount)

	for _, result := range results {
		summary += "\n" + result
	}

	if inactiveCount > 0 {
		summary += "\n\n⚠️ <b>Неактивные хосты автоматически отключены и не будут использоваться для создания VPN.</b>"
	} else {
		summary += "\n\n✅ <b>Все хосты работают корректно!</b>"
	}

	summary += "\n\n🔄 <b>Для повторной проверки используйте кнопку «🔍 Проверить хосты» или команду /check_hosts</b>"

	return p.sendMessageWithAdminButtons(client, chatID, summary)
}

// createUserWithActiveState создает нового пользователя с активным состоянием
func (p *MessageProcessor) createUserWithActiveState(userID int64, username, firstName, lastName string, isBot bool) error {
	// Создаем пользователя в базе данных
	user := &contracts.TelegramUser{
		TelegramID: userID,
		Username:   username,
		FirstName:  firstName,
		LastName:   lastName,
		IsBot:      isBot,
	}

	// Используем UserService для создания пользователя
	if err := p.userService.CreateUser(user); err != nil {
		return fmt.Errorf("ошибка создания пользователя: %w", err)
	}

	// Устанавливаем активное состояние через SQL запрос
	// Это проще, чем работать с адаптерами
	query := `
		UPDATE telegram_users SET
			state = 'active',
			expected_action = 'none',
			state_changed_at = CURRENT_TIMESTAMP,
			state_reason = 'Автоматическое создание пользователя',
			state_changed_by_tg_id = $1,
			state_changed_by_username = $2,
			state_expires_at = NULL,
			state_metadata = '{}',
			updated_at = CURRENT_TIMESTAMP
		WHERE telegram_id = $1
	`

	// Получаем базу данных из UserService
	db := p.userService.GetDB()
	if db == nil {
		return fmt.Errorf("не удалось получить подключение к базе данных")
	}

	result, err := db.Exec(query, userID, username)
	if err != nil {
		return fmt.Errorf("ошибка установки активного состояния: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества обновленных строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("пользователь с Telegram ID %d не найден для обновления состояния", userID)
	}

	return nil
}

// handleVPNCommand обрабатывает команду /vpn
func (p *MessageProcessor) handleVPNCommand(client *TelegramClient, chatID int, userID int64, username string, userState *contracts.UserStateInfo) error {
	// Получаем VPN подключения пользователя
	connections, err := p.vpnConnectionService.GetUserVPNConnections(userID)
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка получения VPN подключений пользователя %d: %v", userID, err)
		return p.sendErrorMessage(client, chatID, "Ошибка получения VPN подключений")
	}

	if len(connections) == 0 {
		message := "🔒 <b>У вас пока нет VPN подключений</b>\n\nДля создания VPN подключения используйте кнопку ниже:"
		return p.sendMessageWithCreateVPNButton(client, chatID, message)
	}

	// Формируем список VPN подключений
	message := fmt.Sprintf("🔒 <b>Ваши VPN подключения (%d)</b>\n\n", len(connections))

	for i, connection := range connections {
		message += fmt.Sprintf("<b>%d. VPN #%d</b>\n", i+1, connection.ID)
		message += fmt.Sprintf("📧 Email: <code>%s</code>\n", connection.Email)
		message += fmt.Sprintf("🔌 Порт: <code>%d</code>\n", connection.Port)
		message += fmt.Sprintf("📅 Создано: %s\n", connection.CreatedAt.Format("02.01.2006 15:04"))
		message += "\n"
	}

	message += "💡 <b>Для получения данных подключения нажмите на соответствующую кнопку ниже</b>"

	return p.sendMessageWithVPNButtons(client, chatID, message, connections)
}

// handleVPNCallback обрабатывает callback-запросы для VPN
func (p *MessageProcessor) handleVPNCallback(client *TelegramClient, chatID int, userID int64, callbackData string) error {
	parts := strings.Split(callbackData, "_")
	if len(parts) < 3 {
		return p.sendErrorMessage(client, chatID, "Неверный формат callback данных")
	}

	action := parts[1]
	connectionID := parts[2]

	// Парсим ID подключения
	var id int
	if _, err := fmt.Sscanf(connectionID, "%d", &id); err != nil {
		return p.sendErrorMessage(client, chatID, "Неверный ID VPN подключения")
	}

	// Получаем VPN подключение
	connection, err := p.vpnConnectionService.GetVPNConnectionByID(id)
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка получения VPN подключения %d: %v", id, err)
		return p.sendErrorMessage(client, chatID, "Ошибка получения VPN подключения")
	}

	if connection == nil {
		return p.sendErrorMessage(client, chatID, "VPN подключение не найдено")
	}

	// Проверяем, что подключение принадлежит пользователю
	if connection.TelegramUserID != userID {
		return p.sendErrorMessage(client, chatID, "Это VPN подключение не принадлежит вам")
	}

	switch action {
	case "info":
		return p.showVPNConnectionInfo(client, chatID, connection)
	case "delete":
		return p.deleteVPNConnection(client, chatID, connection)
	case "refresh":
		return p.handleVPNCommand(client, chatID, userID, "", nil)
	default:
		return p.sendErrorMessage(client, chatID, "Неизвестное действие")
	}
}

// showVPNConnectionInfo показывает информацию о VPN подключении
func (p *MessageProcessor) showVPNConnectionInfo(client *TelegramClient, chatID int, connection *services.VPNConnection) error {
	message := fmt.Sprintf("🔒 <b>VPN подключение #%d</b>\n\n", connection.ID)
	message += fmt.Sprintf("📧 <b>Email:</b> <code>%s</code>\n", connection.Email)
	message += fmt.Sprintf("🔌 <b>Порт:</b> <code>%d</code>\n", connection.Port)
	message += fmt.Sprintf("🆔 <b>Client ID:</b> <code>%s</code>\n", connection.ClientID)
	message += fmt.Sprintf("📅 <b>Создано:</b> %s\n\n", connection.CreatedAt.Format("02.01.2006 15:04:05"))

	message += "🔗 <b>VLESS ссылка для подключения:</b>\n"
	message += fmt.Sprintf("<code>%s</code>\n\n", connection.VlessLink)

	message += "📱 <b>Для подключения:</b>\n"
	message += "1. Скопируйте VLESS ссылку выше\n"
	message += "2. Откройте приложение V2rayNG или аналогичное\n"
	message += "3. Нажмите «+» и выберите «Импорт из буфера обмена»\n"
	message += "4. Вставьте ссылку и нажмите «Сохранить»\n\n"

	message += "⚠️ <b>Важно:</b> Храните эту ссылку в безопасном месте!"

	return p.sendMessageHTML(client, chatID, message)
}

// deleteVPNConnection удаляет VPN подключение
func (p *MessageProcessor) deleteVPNConnection(client *TelegramClient, chatID int, connection *services.VPNConnection) error {
	// Деактивируем подключение в базе данных
	if err := p.vpnConnectionService.DeactivateVPNConnection(connection.ID); err != nil {
		log.Printf("[MessageProcessor] Ошибка деактивации VPN подключения %d: %v", connection.ID, err)
		return p.sendErrorMessage(client, chatID, "Ошибка удаления VPN подключения")
	}

	message := fmt.Sprintf("🗑️ <b>VPN подключение #%d удалено</b>\n\n", connection.ID)
	message += "✅ Подключение успешно деактивировано в базе данных.\n\n"
	message += "💡 Для создания нового VPN подключения используйте команду /vpn"

	return p.sendMessageHTML(client, chatID, message)
}

// sendMessageWithVPNButtons отправляет сообщение с кнопками VPN подключений
func (p *MessageProcessor) sendMessageWithVPNButtons(client *TelegramClient, chatID int, text string, connections []*services.VPNConnection) error {
	var keyboard [][]InlineKeyboardButton

	// Добавляем кнопки для каждого VPN подключения
	for _, connection := range connections {
		row := []InlineKeyboardButton{
			{
				Text:         fmt.Sprintf("ℹ️ VPN #%d", connection.ID),
				CallbackData: fmt.Sprintf("vpn_info_%d", connection.ID),
			},
			{
				Text:         fmt.Sprintf("🗑️ Удалить #%d", connection.ID),
				CallbackData: fmt.Sprintf("vpn_delete_%d", connection.ID),
			},
		}
		keyboard = append(keyboard, row)
	}

	// Добавляем кнопку создания нового VPN
	keyboard = append(keyboard, []InlineKeyboardButton{
		{
			Text:         "➕ Создать новое VPN",
			CallbackData: "create_vpn",
		},
	})

	// Добавляем кнопку обновления списка
	keyboard = append(keyboard, []InlineKeyboardButton{
		{
			Text:         "🔄 Обновить список",
			CallbackData: "vpn_refresh",
		},
	})

	inlineKeyboard := &InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}

	// SendMessageWithKeyboard уже поддерживает HTML форматирование
	_, err := client.SendMessageWithKeyboard(chatID, text, inlineKeyboard)
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка отправки сообщения с VPN кнопками: %v", err)
	}
	return err
}

// handleTransactionsCommand выводит список транзакций для админа
func (p *MessageProcessor) handleTransactionsCommand(client *TelegramClient, chatID int, userID int64) error {
	if !p.adminService.IsGlobalAdmin(userID) {
		return p.sendMessage(client, chatID, "❌ Только глобальные администраторы могут просматривать транзакции.")
	}

	transactions, err := p.transactionService.GetAllTransactions()
	if err != nil {
		log.Printf("[MessageProcessor] Ошибка получения транзакций: %v", err)
		return p.sendErrorMessage(client, chatID, "Ошибка получения транзакций")
	}
	if len(transactions) == 0 {
		return p.sendMessageHTML(client, chatID, "ℹ️ <b>Транзакций не найдено</b>")
	}

	var sb strings.Builder
	sb.WriteString("<b>Последние транзакции:</b>\n\n")
	var keyboard [][]InlineKeyboardButton
	for _, tx := range transactions {
		row := []InlineKeyboardButton{}
		ts := tx.CreatedAt.Format("02.01.06 15:04")
		sb.WriteString(fmt.Sprintf("ID: <code>%d</code> | User: <code>%d</code> | %s\nСумма: <b>%d</b> | Тип: <b>%s</b> | Статус: <b>%s</b>\nПричина: %s\n---\n",
			tx.ID, tx.TelegramUserID, ts, tx.Amount, tx.Type, tx.Status, tx.Reason))
		if tx.Type == "payment" && tx.Status == "success" {
			row = append(row, InlineKeyboardButton{
				Text:         "↩️ Возврат средств",
				CallbackData: fmt.Sprintf("refund_%d", tx.ID),
			})
		}
		if len(row) > 0 {
			keyboard = append(keyboard, row)
		}
	}
	inlineKeyboard := &InlineKeyboardMarkup{InlineKeyboard: keyboard}
	_, err = client.SendMessageWithKeyboard(chatID, sb.String(), inlineKeyboard)
	return err
}

// handleRefundCallback обрабатывает возврат средств по транзакции
func (p *MessageProcessor) handleRefundCallback(client *TelegramClient, chatID int, userID int64, callbackData string) error {
	if !p.adminService.IsGlobalAdmin(userID) {
		return p.sendErrorMessage(client, chatID, "Нет прав для возврата средств")
	}
	parts := strings.Split(callbackData, "_")
	if len(parts) != 2 {
		return p.sendErrorMessage(client, chatID, "Неверный формат callback для возврата")
	}
	var txID int
	if _, err := fmt.Sscanf(parts[1], "%d", &txID); err != nil {
		return p.sendErrorMessage(client, chatID, "Неверный ID транзакции")
	}
	// Получаем транзакцию
	transactions, err := p.transactionService.GetAllTransactions()
	if err != nil {
		return p.sendErrorMessage(client, chatID, "Ошибка поиска транзакции")
	}
	var tx *services.Transaction
	for _, t := range transactions {
		if t.ID == txID {
			tx = t
			break
		}
	}
	if tx == nil {
		return p.sendErrorMessage(client, chatID, "Транзакция не найдена")
	}
	if tx.Type != "payment" || tx.Status != "success" {
		return p.sendErrorMessage(client, chatID, "Возврат возможен только для успешных платежей")
	}
	if tx.TelegramPaymentChargeID == "" {
		return p.sendErrorMessage(client, chatID, "В транзакции отсутствует идентификатор платежа (telegram_payment_charge_id). Возврат невозможен.")
	}
	// Проверяем, что это не тестовый платёж
	if tx.InvoicePayload != "" && (tx.InvoicePayload == "test" || tx.InvoicePayload == "vpn_create_test") {
		return p.sendErrorMessage(client, chatID, "Возврат невозможен для тестовых платежей Telegram Stars.")
	}
	// Логируем детали транзакции для диагностики
	log.Printf("[DEBUG] Refund: txID=%d, userID=%d, chargeID=%s, amount=%d, status=%s, type=%s, payload=%s",
		tx.ID, tx.TelegramUserID, tx.TelegramPaymentChargeID, tx.Amount, tx.Status, tx.Type, tx.InvoicePayload)
	// Делаем возврат
	errRefund := client.RefundStarPayment(tx.TelegramUserID, tx.TelegramPaymentChargeID, tx.Amount, "Возврат по запросу админа")
	if errRefund != nil {
		return p.sendErrorMessage(client, chatID, fmt.Sprintf("Ошибка возврата: %v", errRefund))
	}
	// Записываем транзакцию возврата
	refundTx := &services.Transaction{
		TelegramPaymentChargeID: tx.TelegramPaymentChargeID,
		TelegramUserID:          tx.TelegramUserID,
		Amount:                  tx.Amount,
		InvoicePayload:          tx.InvoicePayload,
		Status:                  "success",
		Type:                    "refund",
		Reason:                  "Возврат по запросу админа",
	}
	_ = p.transactionService.AddTransaction(refundTx)
	return p.sendMessageHTML(client, chatID, "✅ Возврат средств инициирован через Telegram Stars")
}

package telegram

import (
	"fmt"
	"log"
	"strings"
)

// routeMessage - основной маршрутизатор команд и событий
func (p *MessageProcessor) routeMessage(client *TelegramClient, update Update) error {
	if update.PreCheckoutQuery != nil {
		return p.handlePreCheckout(client, update)
	}
	if update.Message != nil && update.Message.SuccessfulPayment != nil {
		return p.handleSuccessfulPayment(client, update)
	}
	if update.CallbackQuery != nil {
		return p.handleCallback(client, update)
	}
	if update.Message != nil && update.Message.Text != "" {
		return p.handleTextMessage(client, update)
	}
	log.Printf("[routeMessage] Неизвестный тип update: %+v", update)
	return nil
}

// handleTextMessage - обработка текстовых сообщений
func (p *MessageProcessor) handleTextMessage(client *TelegramClient, update Update) error {
	messageText := update.Message.Text
	if len(messageText) > 0 && messageText[0] == '/' {
		return p.handleCommand(client, update)
	}
	return p.handleUserStateMessage(client, update)
}

// handleCommand - обработка команд Telegram
func (p *MessageProcessor) handleCommand(client *TelegramClient, update Update) error {
	command := update.Message.Text
	switch command {
	case "/start":
		return p.handleStartCommand(client, update)
	case "/help":
		return p.handleHelpCommand(client, update)
	case "/cancel":
		return p.handleCancelCommand(client, update)
	case "/addhost":
		return p.handleAddHostCommand(client, update)
	case "/monitor":
		return p.handleMonitorCommand(client, update)
	case "/monitor_start":
		return p.handleMonitorStartCommand(client, update)
	case "/monitor_stop":
		return p.handleMonitorStopCommand(client, update)
	case "/monitor_status":
		return p.handleMonitorStatusCommand(client, update)
	case "/check_hosts":
		return p.handleCheckHostsCommand(client, update)
	case "/transactions":
		return p.handleTransactionsCommand(client, update)
	// ... другие команды ...
	default:
		return p.sendMessage(client, update.Message.Chat.ID, "Неизвестная команда. Используйте /help для справки.")
	}
}

func (p *MessageProcessor) handleTransactionsCommand(client *TelegramClient, update Update) error {
	userID := int64(update.Message.From.ID)
	if !p.adminService.IsGlobalAdmin(userID) {
		return p.sendMessage(client, update.Message.Chat.ID, "❌ Только глобальные администраторы могут просматривать транзакции.")
	}
	transactions, err := p.transactionService.GetAllTransactions()
	if err != nil {
		return p.sendErrorMessage(client, update.Message.Chat.ID, "Ошибка получения транзакций")
	}
	if len(transactions) == 0 {
		return p.sendMessageHTML(client, update.Message.Chat.ID, "ℹ️ <b>Транзакций не найдено</b>")
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
	_, err = client.SendMessageWithKeyboard(update.Message.Chat.ID, sb.String(), inlineKeyboard)
	return err
}

// handleUserStateMessage - обработка сообщений по состоянию пользователя
func (p *MessageProcessor) handleUserStateMessage(client *TelegramClient, update Update) error {
	userID := int64(update.Message.From.ID)
	userState, err := p.userStateService.GetUserState(userID)
	if err != nil {
		return p.sendErrorMessage(client, update.Message.Chat.ID, "Ошибка получения состояния пользователя")
	}
	if userState == nil {
		return p.sendErrorMessage(client, update.Message.Chat.ID, "Пользователь не найден в системе")
	}
	// TODO: здесь может быть делегирование в обработчики по состоянию
	return p.sendMessageHTML(client, update.Message.Chat.ID, "Ваше текущее состояние: <b>"+userState.State+"</b>")
}

// handleStartCommand, handleHelpCommand, handleCancelCommand, handleAddHostCommand, handleMonitorCommand — заготовки для дальнейшей реализации
func (p *MessageProcessor) handleStartCommand(client *TelegramClient, update Update) error {
	user := update.Message.From
	userID := int64(user.ID)
	message := "🤖 <b>Добро пожаловать в TelegramXUI!</b>\n\n" +
		"👤 <b>Пользователь:</b> " + user.Username + "\n" +
		"✅ <b>Регистрация:</b> Автоматически завершена\n" +
		"🎯 <b>Доступ:</b> Полный доступ к функциям\n\n" +
		"Используйте /help для получения справки."
	var keyboard *InlineKeyboardMarkup
	if p.adminService.IsGlobalAdmin(userID) {
		keyboard = makeAdminButtons()
	} else {
		keyboard = makeCreateVPNButton()
	}
	return p.sendMessageWithKeyboard(client, update.Message.Chat.ID, message, keyboard)
}
func (p *MessageProcessor) handleHelpCommand(client *TelegramClient, update Update) error {
	userID := int64(update.Message.From.ID)
	message := "📚 <b>Справка по командам:</b>\n" +
		"/start - Начать работу с ботом\n" +
		"/help - Показать эту справку\n" +
		"/cancel - Отменить текущую операцию\n" +
		"/vpn - Управление VPN подключениями\n"
	if p.adminService.IsGlobalAdmin(userID) {
		message += "/addhost - Добавить XUI хост\n/monitor - Управление мониторингом хостов\n/monitor_start - Запустить мониторинг\n/monitor_stop - Остановить мониторинг\n/monitor_status - Статус мониторинга\n/check_hosts - Проверить все хосты сейчас\n/transactions - Просмотр транзакций\n"
	}
	message += "\n💡 <b>Совет:</b> Нажмите кнопку 'Создать VPN' для быстрого доступа к VPN"
	var keyboard *InlineKeyboardMarkup
	if p.adminService.IsGlobalAdmin(userID) {
		keyboard = makeAdminButtons()
	} else {
		keyboard = makeCreateVPNButton()
	}
	return p.sendMessageWithKeyboard(client, update.Message.Chat.ID, message, keyboard)
}
func (p *MessageProcessor) handleCancelCommand(client *TelegramClient, update Update) error {
	// TODO: здесь может быть сброс состояния пользователя
	return p.sendMessageHTML(client, update.Message.Chat.ID, "✅ Процесс отменён. Вы вернулись в обычное состояние.")
}
func (p *MessageProcessor) handleAddHostCommand(client *TelegramClient, update Update) error {
	// TODO: здесь может быть запуск процесса добавления хоста
	return p.sendMessageHTML(client, update.Message.Chat.ID, "📝 Введите данные нового XUI хоста в формате: <code>хост логин пароль [секретный_ключ]</code>")
}
func (p *MessageProcessor) handleMonitorCommand(client *TelegramClient, update Update) error {
	userID := int64(update.Message.From.ID)
	if !p.adminService.IsGlobalAdmin(userID) {
		return p.sendMessage(client, update.Message.Chat.ID, "❌ Только глобальные администраторы могут управлять мониторингом.")
	}
	status := p.hostMonitorService.GetMonitoringStatus()
	isRunning, _ := status["is_running"].(bool)
	interval, _ := status["check_interval"].(string)
	servers, err := p.xuiServerService.GetAllServers(1000, 0)
	if err != nil {
		return p.sendErrorMessage(client, update.Message.Chat.ID, "Ошибка получения списка хостов")
	}
	active, inactive := 0, 0
	for _, s := range servers {
		if s.IsActive {
			active++
		} else {
			inactive++
		}
	}
	total := len(servers)
	statusStr := "🔴 Остановлен"
	if isRunning {
		statusStr = "🟢 Запущен"
	}
	msg := fmt.Sprintf("🔍 <b>Мониторинг хостов</b>\n\n📊 Статус: %s\n⏱️ Интервал проверки: <b>%s</b>\n\n📈 Хостов всего: <b>%d</b>\n🟢 Активных: <b>%d</b>\n🔴 Неактивных: <b>%d</b>\n\nКоманды:\n/monitor_start — Запустить\n/monitor_stop — Остановить\n/monitor_status — Статус\n/check_hosts — Проверить все сейчас", statusStr, interval, total, active, inactive)
	return p.sendMessageHTML(client, update.Message.Chat.ID, msg)
}

func (p *MessageProcessor) handleMonitorStartCommand(client *TelegramClient, update Update) error {
	userID := int64(update.Message.From.ID)
	if !p.adminService.IsGlobalAdmin(userID) {
		return p.sendMessage(client, update.Message.Chat.ID, "❌ Только глобальные администраторы могут запускать мониторинг.")
	}
	err := p.hostMonitorService.Start()
	if err != nil {
		return p.sendErrorMessage(client, update.Message.Chat.ID, "Ошибка запуска мониторинга: "+err.Error())
	}
	return p.sendMessageHTML(client, update.Message.Chat.ID, "🟢 Мониторинг хостов запущен!")
}

func (p *MessageProcessor) handleMonitorStopCommand(client *TelegramClient, update Update) error {
	userID := int64(update.Message.From.ID)
	if !p.adminService.IsGlobalAdmin(userID) {
		return p.sendMessage(client, update.Message.Chat.ID, "❌ Только глобальные администраторы могут останавливать мониторинг.")
	}
	err := p.hostMonitorService.Stop()
	if err != nil {
		return p.sendErrorMessage(client, update.Message.Chat.ID, "Ошибка остановки мониторинга: "+err.Error())
	}
	return p.sendMessageHTML(client, update.Message.Chat.ID, "🔴 Мониторинг хостов остановлен!")
}

func (p *MessageProcessor) handleMonitorStatusCommand(client *TelegramClient, update Update) error {
	userID := int64(update.Message.From.ID)
	if !p.adminService.IsGlobalAdmin(userID) {
		return p.sendMessage(client, update.Message.Chat.ID, "❌ Только глобальные администраторы могут просматривать статус мониторинга.")
	}
	status := p.hostMonitorService.GetMonitoringStatus()
	isRunning, _ := status["is_running"].(bool)
	interval, _ := status["check_interval"].(string)
	servers, err := p.xuiServerService.GetAllServers(1000, 0)
	if err != nil {
		return p.sendErrorMessage(client, update.Message.Chat.ID, "Ошибка получения списка хостов")
	}
	active, inactive := 0, 0
	var inactiveList []string
	for _, s := range servers {
		if s.IsActive {
			active++
		} else {
			inactive++
			inactiveList = append(inactiveList, fmt.Sprintf("• %s (%s)", s.ServerName, s.ServerURL))
		}
	}
	total := len(servers)
	statusStr := "🔴 Остановлен"
	if isRunning {
		statusStr = "🟢 Запущен"
	}
	msg := fmt.Sprintf("📊 <b>Статус мониторинга хостов</b>\n\n🔄 Мониторинг: %s\n⏱️ Интервал: <b>%s</b>\n\n📈 Всего: <b>%d</b>\n🟢 Активных: <b>%d</b>\n🔴 Неактивных: <b>%d</b>", statusStr, interval, total, active, inactive)
	if inactive > 0 {
		msg += "\n\n⚠️ Неактивные хосты:\n" + strings.Join(inactiveList, "\n")
	}
	return p.sendMessageHTML(client, update.Message.Chat.ID, msg)
}

func (p *MessageProcessor) handleCheckHostsCommand(client *TelegramClient, update Update) error {
	userID := int64(update.Message.From.ID)
	if !p.adminService.IsGlobalAdmin(userID) {
		return p.sendMessage(client, update.Message.Chat.ID, "❌ Только глобальные администраторы могут проверять хосты.")
	}
	go p.hostMonitorService.CheckAllHosts() // Запускаем асинхронно, чтобы не блокировать бота
	return p.sendMessageHTML(client, update.Message.Chat.ID, "🔍 Начинаем проверку всех хостов... Результаты придут в личку админа после завершения!")
}

// sendMessageWithKeyboard отправляет сообщение с inline-клавиатурой
func (p *MessageProcessor) sendMessageWithKeyboard(client *TelegramClient, chatID int, text string, keyboard *InlineKeyboardMarkup) error {
	_, err := client.SendMessageWithKeyboard(chatID, text, keyboard)
	return err
}

package telegram

import (
	"TelegramXUI/internal/services"
	"TelegramXUI/internal/xui_client"
	"fmt"
	"log"
	"time"
)

// handlePreCheckout - обработка pre_checkout_query
func (p *MessageProcessor) handlePreCheckout(client *TelegramClient, update Update) error {
	log.Printf("[handlePreCheckout] pre_checkout_query: id=%s, user_id=%d, currency=%s, total_amount=%d, payload=%s",
		update.PreCheckoutQuery.ID,
		update.PreCheckoutQuery.From.ID,
		update.PreCheckoutQuery.Currency,
		update.PreCheckoutQuery.TotalAmount,
		update.PreCheckoutQuery.InvoicePayload)
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

// handleSuccessfulPayment - обработка успешного платежа
func (p *MessageProcessor) handleSuccessfulPayment(client *TelegramClient, update Update) error {
	log.Printf("[handleSuccessfulPayment] Получен SuccessfulPayment: %+v", update)
	if update.Message.From.ID == 0 {
		log.Printf("[ERROR] update.Message.SuccessfulPayment, но From.ID == 0: %+v", update)
		return p.sendErrorMessage(client, 0, "Ошибка: не удалось определить пользователя для оплаты")
	}
	userID := int64(update.Message.From.ID)
	chatID := update.Message.Chat.ID
	log.Printf("[handleSuccessfulPayment] userID=%d, chatID=%d", userID, chatID)

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

func (p *MessageProcessor) createVPNAndSendInfo(client *TelegramClient, chatID int, userID int64) error {
	user, err := p.userService.GetUserByTelegramID(userID)
	if err != nil {
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

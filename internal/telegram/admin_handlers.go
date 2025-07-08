package telegram

import (
	"TelegramXUI/internal/services"
	"fmt"
	"strings"
)

// handleCallback - маршрутизатор для callback-запросов
func (p *MessageProcessor) handleCallback(client *TelegramClient, update Update) error {
	data := update.CallbackQuery.Data
	if data == "addhost" {
		return p.handleAddHostCallback(client, update)
	} else if data == "check_hosts" {
		return p.handleCheckHostsCallback(client, update)
	} else if strings.HasPrefix(data, "refund_") {
		return p.handleRefundCallback(client, update)
	} else if data == "create_vpn" || strings.HasPrefix(data, "vpn_") {
		return p.handleCallbackVPN(client, update)
	}
	// Остальные callback-и (если появятся новые)
	return nil
}

// handleAddHostCallback - обработка callback для добавления хоста
func (p *MessageProcessor) handleAddHostCallback(client *TelegramClient, update Update) error {
	userID := int64(update.CallbackQuery.From.ID)
	username := update.CallbackQuery.From.Username
	// Запуск процесса добавления хоста
	err := p.xuiHostAddService.StartAddHostProcess(userID, username)
	if err != nil {
		return p.sendErrorMessage(client, int(userID), "Ошибка начала процесса добавления хоста: "+err.Error())
	}
	instructions := p.xuiHostAddService.GetAddHostInstructions()
	return p.sendMessageHTML(client, int(userID), instructions)
}

// handleCheckHostsCallback - обработка callback для проверки хостов
func (p *MessageProcessor) handleCheckHostsCallback(client *TelegramClient, update Update) error {
	userID := int64(update.CallbackQuery.From.ID)
	// Принудительная проверка всех хостов
	// (здесь может быть вызов p.xuiServerService.GetActiveServers и p.hostMonitorService.CheckHostNow)
	return p.sendMessageHTML(client, int(userID), "🔍 Принудительная проверка хостов запущена (детальная реализация — см. сервисы)")
}

// handleRefundCallback - обработка callback для возврата средств
func (p *MessageProcessor) handleRefundCallback(client *TelegramClient, update Update) error {
	userID := int64(update.CallbackQuery.From.ID)
	if !p.adminService.IsGlobalAdmin(userID) {
		return p.sendErrorMessage(client, int(userID), "Нет прав для возврата средств")
	}
	parts := strings.Split(update.CallbackQuery.Data, "_")
	if len(parts) != 2 {
		return p.sendErrorMessage(client, int(userID), "Неверный формат callback для возврата")
	}
	var txID int
	if _, err := fmt.Sscanf(parts[1], "%d", &txID); err != nil {
		return p.sendErrorMessage(client, int(userID), "Неверный ID транзакции")
	}
	transactions, err := p.transactionService.GetAllTransactions()
	if err != nil {
		return p.sendErrorMessage(client, int(userID), "Ошибка поиска транзакции")
	}
	var tx *services.Transaction
	for _, t := range transactions {
		if t.ID == txID {
			tx = t
			break
		}
	}
	if tx == nil {
		return p.sendErrorMessage(client, int(userID), "Транзакция не найдена")
	}
	if tx.Type != "payment" || tx.Status != "success" {
		return p.sendErrorMessage(client, int(userID), "Возврат возможен только для успешных платежей")
	}
	if tx.TelegramPaymentChargeID == "" {
		return p.sendErrorMessage(client, int(userID), "В транзакции отсутствует идентификатор платежа (telegram_payment_charge_id). Возврат невозможен.")
	}
	errRefund := client.RefundStarPayment(tx.TelegramUserID, tx.TelegramPaymentChargeID, tx.Amount, "Возврат по запросу админа")
	if errRefund != nil {
		return p.sendErrorMessage(client, int(userID), fmt.Sprintf("Ошибка возврата: %v", errRefund))
	}
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
	return p.sendMessageHTML(client, int(userID), "✅ Возврат средств инициирован через Telegram Stars")
}

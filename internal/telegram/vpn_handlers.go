package telegram

import (
	"fmt"
	"strings"
)

// handleCallbackVPN - маршрутизатор для VPN callback-запросов
func (p *MessageProcessor) handleCallbackVPN(client *TelegramClient, update Update) error {
	data := update.CallbackQuery.Data
	if data == "create_vpn" {
		return p.handleCreateVPNCallback(client, update)
	} else if strings.HasPrefix(data, "vpn_info_") {
		return p.handleVPNInfoCallback(client, update)
	} else if strings.HasPrefix(data, "vpn_delete_") {
		return p.handleVPNDeleteCallback(client, update)
	} else if data == "vpn_refresh" {
		return p.handleVPNRefreshCallback(client, update)
	}
	return nil
}

// handleCreateVPNCallback - обработка callback для создания VPN
func (p *MessageProcessor) handleCreateVPNCallback(client *TelegramClient, update Update) error {
	chatID := int(update.CallbackQuery.From.ID)
	title := "Создание VPN-подключения"
	description := "Оплата за создание VPN через Telegram Stars"
	payload := fmt.Sprintf("vpn_create_%d", update.CallbackQuery.From.ID)
	providerToken := "" // Для Stars provider_token пустой
	currency := "XTR"
	prices := []LabeledPrice{{Label: "VPN", Amount: 1}}
	isTest := false
	return client.SendInvoice(chatID, title, description, payload, providerToken, currency, prices, isTest)
}

// handleVPNInfoCallback - обработка callback для информации о VPN
func (p *MessageProcessor) handleVPNInfoCallback(client *TelegramClient, update Update) error {
	// Извлекаем ID VPN из callback data
	var vpnID int
	_, err := fmt.Sscanf(update.CallbackQuery.Data, "vpn_info_%d", &vpnID)
	if err != nil {
		return p.sendErrorMessage(client, int(update.CallbackQuery.From.ID), "Неверный формат callback данных")
	}
	// Получаем VPN подключение
	vpn, err := p.vpnConnectionService.GetVPNConnectionByID(vpnID)
	if err != nil || vpn == nil {
		return p.sendErrorMessage(client, int(update.CallbackQuery.From.ID), "VPN подключение не найдено")
	}
	// Отправляем информацию
	msg := fmt.Sprintf("🔒 <b>VPN #%d</b>\n📧 Email: <code>%s</code>\n🔌 Порт: <code>%d</code>\n🆔 Client ID: <code>%s</code>\n\n<code>%s</code>", vpn.ID, vpn.Email, vpn.Port, vpn.ClientID, vpn.VlessLink)
	return p.sendMessageHTML(client, int(update.CallbackQuery.From.ID), msg)
}

// handleVPNDeleteCallback - обработка callback для удаления VPN
func (p *MessageProcessor) handleVPNDeleteCallback(client *TelegramClient, update Update) error {
	var vpnID int
	_, err := fmt.Sscanf(update.CallbackQuery.Data, "vpn_delete_%d", &vpnID)
	if err != nil {
		return p.sendErrorMessage(client, int(update.CallbackQuery.From.ID), "Неверный формат callback данных")
	}
	err = p.vpnConnectionService.DeactivateVPNConnection(vpnID)
	if err != nil {
		return p.sendErrorMessage(client, int(update.CallbackQuery.From.ID), "Ошибка удаления VPN: "+err.Error())
	}
	return p.sendMessageHTML(client, int(update.CallbackQuery.From.ID), "🗑️ VPN подключение удалено")
}

// handleVPNRefreshCallback - обработка callback для обновления списка VPN
func (p *MessageProcessor) handleVPNRefreshCallback(client *TelegramClient, update Update) error {
	userID := int64(update.CallbackQuery.From.ID)
	connections, err := p.vpnConnectionService.GetUserVPNConnections(userID)
	if err != nil {
		return p.sendErrorMessage(client, int(userID), "Ошибка получения VPN подключений")
	}
	msg := fmt.Sprintf("🔒 <b>Ваши VPN подключения (%d)</b>", len(connections))
	return p.sendMessageHTML(client, int(userID), msg)
}

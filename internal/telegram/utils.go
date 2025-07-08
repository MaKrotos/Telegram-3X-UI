package telegram

import (
	"log"
)

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
	errorMessage := "❌ <b>Ошибка:</b> " + text
	return p.sendMessageHTML(client, chatID, errorMessage)
}

// makeCreateVPNButton возвращает inline-клавиатуру с кнопкой "Создать VPN"
func makeCreateVPNButton() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{
					Text:         "🔑 Создать VPN",
					CallbackData: "create_vpn",
				},
			},
		},
	}
}

// makeAdminButtons возвращает inline-клавиатуру с админскими кнопками
func makeAdminButtons() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "➕ Добавить хост", CallbackData: "admin_addhost"},
				{Text: "🖥 Мониторинг", CallbackData: "admin_monitor"},
			},
			{
				{Text: "🔍 Проверить хосты", CallbackData: "admin_check_hosts"},
				{Text: "💸 Транзакции", CallbackData: "admin_transactions"},
			},
		},
	}
}

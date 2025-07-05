package contracts

// TelegramMessageSender интерфейс для отправки сообщений в Telegram
type TelegramMessageSender interface {
	SendMessage(chatID int64, message string) error
}

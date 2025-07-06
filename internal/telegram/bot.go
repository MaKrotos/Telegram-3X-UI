package telegram

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// BotMode определяет режим работы бота
type BotMode string

const (
	ModePolling BotMode = "polling"
	ModeWebhook BotMode = "webhook"
)

// TelegramBot представляет универсального бота с поддержкой polling и webhook
type TelegramBot struct {
	client     *TelegramClient
	mode       BotMode
	webhookURL string
	server     *http.Server
	mu         sync.Mutex
	handlers   []MessageHandler
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// MessageHandler функция для обработки сообщений
type MessageHandler func(client *TelegramClient, update Update) error

// BotConfig конфигурация бота
type BotConfig struct {
	Token      string
	Mode       BotMode
	WebhookURL string
	Port       int
}

// NewBot создает нового бота с указанной конфигурацией
func NewBot(config BotConfig) (*TelegramBot, error) {
	if config.Token == "" {
		return nil, fmt.Errorf("токен бота не может быть пустым")
	}

	client := NewClient(config.Token)

	// Проверяем токен
	_, err := client.GetMe()
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки токена: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	bot := &TelegramBot{
		client:     client,
		mode:       config.Mode,
		webhookURL: config.WebhookURL,
		ctx:        ctx,
		cancel:     cancel,
		handlers:   make([]MessageHandler, 0),
	}

	if config.Mode == ModeWebhook && config.WebhookURL == "" {
		return nil, fmt.Errorf("webhook URL обязателен для webhook режима")
	}

	return bot, nil
}

// AddHandler добавляет обработчик сообщений
func (b *TelegramBot) AddHandler(handler MessageHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, handler)
}

// Start запускает бота в указанном режиме
func (b *TelegramBot) Start() error {
	log.Printf("[TelegramBot] Запуск Telegram бота в режиме: %s", b.mode)

	switch b.mode {
	case ModePolling:
		return b.startPolling()
	case ModeWebhook:
		return b.startWebhook()
	default:
		return fmt.Errorf("неизвестный режим: %s", b.mode)
	}
}

// Stop останавливает бота
func (b *TelegramBot) Stop() error {
	log.Println("Остановка Telegram бота...")

	b.cancel()

	if b.server != nil {
		if err := b.server.Shutdown(context.Background()); err != nil {
			log.Printf("Ошибка остановки HTTP сервера: %v", err)
		}
	}

	b.wg.Wait()
	log.Println("Telegram бот остановлен")
	return nil
}

// startPolling запускает бота в режиме polling
func (b *TelegramBot) startPolling() error {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()

		offset := 0
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		log.Println("[TelegramBot] Polling режим запущен")
		for {
			select {
			case <-b.ctx.Done():
				log.Println("[TelegramBot] Polling остановлен")
				return
			case <-ticker.C:
				updates, err := b.client.GetUpdates(offset, 10)
				if err != nil {
					log.Printf("[TelegramBot] Ошибка получения обновлений: %v", err)
					continue
				}

				for _, update := range updates.Result {
					var logText string
					var userID int
					if update.Message != nil {
						logText = update.Message.Text
						userID = update.Message.From.ID
					} else if update.CallbackQuery != nil {
						logText = "callback_query"
						userID = update.CallbackQuery.From.ID
					} else {
						logText = "unknown"
						userID = 0
					}
					log.Printf("[TelegramBot] Получен апдейт: update_id=%d, user_id=%d, type=\"%s\"", update.UpdateID, userID, logText)
					b.handleUpdate(update)
					offset = update.UpdateID + 1
				}
			}
		}
	}()

	return nil
}

// startWebhook запускает бота в режиме webhook
func (b *TelegramBot) startWebhook() error {
	// Удаляем старый webhook если есть
	if _, err := b.client.DeleteWebhook(); err != nil {
		log.Printf("Предупреждение: не удалось удалить старый webhook: %v", err)
	}

	// Устанавливаем новый webhook
	result, err := b.client.SetWebhook(b.webhookURL)
	if err != nil {
		return fmt.Errorf("ошибка установки webhook: %w", err)
	}

	// Проверяем результат
	if result["ok"] != true {
		return fmt.Errorf("ошибка установки webhook: %v", result)
	}

	log.Printf("Webhook установлен: %s", b.webhookURL)

	// Создаем HTTP сервер
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", b.webhookHandler)

	port := 8080
	if envPort := os.Getenv("TELEGRAM_WEBHOOK_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		}
	}

	b.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()

		log.Printf("[TelegramBot] Webhook сервер запущен на порту %d", port)
		if err := b.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Ошибка HTTP сервера: %v", err)
		}
	}()

	return nil
}

// webhookHandler обрабатывает входящие webhook запросы
func (b *TelegramBot) webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	update, err := b.client.ParseUpdate(r.Body)
	if err != nil {
		log.Printf("Ошибка парсинга webhook: %v", err)
		http.Error(w, "Ошибка парсинга", http.StatusBadRequest)
		return
	}

	b.handleUpdate(*update)
	w.WriteHeader(http.StatusOK)
}

// handleUpdate обрабатывает обновление
func (b *TelegramBot) handleUpdate(update Update) {
	b.mu.Lock()
	handlers := make([]MessageHandler, len(b.handlers))
	copy(handlers, b.handlers)
	b.mu.Unlock()

	for _, handler := range handlers {
		var logText string
		var userID int
		if update.Message != nil {
			logText = update.Message.Text
			userID = update.Message.From.ID
		} else if update.CallbackQuery != nil {
			logText = "callback_query: " + update.CallbackQuery.Data
			userID = update.CallbackQuery.From.ID
		} else {
			logText = "unknown"
			userID = 0
		}
		log.Printf("[TelegramBot] Вызов обработчика для user_id=%d, type=\"%s\"", userID, logText)
		if err := handler(b.client, update); err != nil {
			log.Printf("[TelegramBot] Ошибка обработчика: %v", err)
		}
	}
}

// SendMessage отправляет сообщение
func (b *TelegramBot) SendMessage(chatID int, text string) error {
	_, err := b.client.SendMessage(chatID, text, "")
	return err
}

// SendMessageHTML отправляет HTML сообщение
func (b *TelegramBot) SendMessageHTML(chatID int, text string) error {
	_, err := b.client.SendMessageHTML(chatID, text)
	return err
}

// GetClient возвращает клиент для прямого доступа к API
func (b *TelegramBot) GetClient() *TelegramClient {
	return b.client
}

// GetMode возвращает текущий режим работы
func (b *TelegramBot) GetMode() BotMode {
	return b.mode
}

// GetWebhookInfo получает информацию о webhook
func (b *TelegramBot) GetWebhookInfo() (map[string]interface{}, error) {
	return b.client.GetWebhookInfo()
}

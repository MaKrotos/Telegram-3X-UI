package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"TelegramXUI/internal/telegram"
	"TelegramXUI/internal/xui_client"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// TelegramBotInstance глобальная переменная для доступа к боту
var TelegramBotInstance *telegram.TelegramBot

// XUIClientInstance глобальная переменная для доступа к x-ui клиенту
var XUIClientInstance *xui_client.Client

// UserServiceInstance глобальная переменная для доступа к сервису пользователей
var UserServiceInstance *telegram.UserService

// WebAppURL URL для WebApp
var WebAppURL = getWebAppURL()

func getWebAppURL() string {
	if url := os.Getenv("TELEGRAM_WEBAPP_URL"); url != "" {
		return url
	}
	return "http://37.46.19.85:5173/" // значение по умолчанию
}

func getDSN() string {
	// Сначала проверяем переменную окружения
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		return dsn
	}

	// Если переменная не задана, формируем DSN из отдельных параметров
	host := getEnvOrDefault("POSTGRES_HOST", "localhost")
	port := getEnvOrDefault("POSTGRES_PORT", "5432")
	user := getEnvOrDefault("POSTGRES_USER", "telegramxui_user")
	password := getEnvOrDefault("POSTGRES_PASSWORD", "telegramxui_password")
	dbname := getEnvOrDefault("POSTGRES_DB", "telegramxui")

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getUsersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Попытка подключения к x-ui
		var xuiStatus string
		if XUIClientInstance != nil {
			if err := XUIClientInstance.Login(); err != nil {
				xuiStatus = "Ошибка подключения к x-ui: " + err.Error()
			} else {
				xuiStatus = "Успешное подключение к x-ui"
			}
		} else {
			xuiStatus = "x-ui клиент не инициализирован"
		}

		// Получаем пользователей из базы данных
		var users []User
		if UserServiceInstance != nil {
			telegramUsers, err := UserServiceInstance.GetAllUsers()
			if err != nil {
				log.Printf("Ошибка получения пользователей: %v", err)
			} else {
				for _, tu := range telegramUsers {
					users = append(users, User{
						ID:   int(tu.TelegramID),
						Name: tu.FirstName + " " + tu.LastName,
					})
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"xui_status": xuiStatus,
			"users":      users,
			"webapp_url": WebAppURL,
		})
	}
}

func getTelegramUsersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if UserServiceInstance == nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Сервис пользователей не инициализирован",
			})
			return
		}

		users, err := UserServiceInstance.GetAllUsers()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Ошибка получения пользователей: " + err.Error(),
			})
			return
		}

		count, err := UserServiceInstance.GetUsersCount()
		if err != nil {
			log.Printf("Ошибка получения количества пользователей: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"users":       users,
			"total_count": count,
			"webapp_url":  WebAppURL,
		})
	}
}

// handleTelegramMessage обрабатывает сообщения от Telegram
func handleTelegramMessage(client *telegram.TelegramClient, update telegram.Update) error {
	if update.Message.Text == "" {
		return nil
	}

	chatID := update.Message.Chat.ID
	text := strings.ToLower(strings.TrimSpace(update.Message.Text))
	userName := update.Message.From.FirstName

	log.Printf("[MessageHandler] Обработка сообщения от %s (ID: %d): %s", userName, chatID, update.Message.Text)

	// Проверяем/создаем пользователя в базе данных
	if UserServiceInstance != nil {
		user, err := UserServiceInstance.EnsureUserExists(update.Message.From)
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
		log.Printf("[MessageHandler] Выполняется команда /start для пользователя %s (ID: %d)", userName, chatID)
		// Отправляем приветственное сообщение с WebApp кнопкой
		_, err := client.SendWelcomeMessageWithWebApp(chatID, userName, WebAppURL)
		if err != nil {
			log.Printf("[MessageHandler] Ошибка отправки приветственного сообщения: %v", err)
			// Fallback на обычное сообщение
			message := fmt.Sprintf("Привет, %s! 👋\n\nЯ бот для управления VPN через x-ui.\n\nДоступные команды:\n/status - Статус x-ui\n/help - Помощь\n/webapp - Открыть панель управления", userName)
			log.Printf("[MessageHandler] Отправка fallback сообщения для пользователя %s (ID: %d)", userName, chatID)
			return TelegramBotInstance.SendMessageHTML(chatID, message)
		}
		log.Printf("[MessageHandler] Приветственное сообщение с WebApp отправлено успешно для пользователя %s (ID: %d)", userName, chatID)
		return nil

	case text == "/webapp" || text == "webapp":
		log.Printf("[MessageHandler] Выполняется команда /webapp для пользователя %s (ID: %d)", userName, chatID)
		// Отправляем WebApp кнопку
		message := "🚀 Нажмите кнопку ниже, чтобы открыть панель управления VPN:"
		_, err := client.SendMessageWithWebAppButton(chatID, message, "📱 Открыть панель", WebAppURL)
		if err != nil {
			log.Printf("[MessageHandler] Ошибка отправки WebApp кнопки: %v", err)
			return TelegramBotInstance.SendMessage(chatID, "Ошибка отправки кнопки. Попробуйте позже.")
		}
		log.Printf("[MessageHandler] WebApp кнопка отправлена успешно для пользователя %s (ID: %d)", userName, chatID)
		return nil

	case text == "/help" || text == "help":
		log.Printf("[MessageHandler] Выполняется команда /help для пользователя %s (ID: %d)", userName, chatID)
		message := `<b>Доступные команды:</b>

/start - Начать работу с ботом
/webapp - Открыть панель управления VPN
/status - Проверить статус x-ui
/users - Статистика пользователей
/help - Показать эту справку

<i>Разработка в процессе...</i>`
		log.Printf("[MessageHandler] Отправка справки для пользователя %s (ID: %d)", userName, chatID)
		return TelegramBotInstance.SendMessageHTML(chatID, message)

	case text == "/status" || text == "status":
		log.Printf("[MessageHandler] Выполняется команда /status для пользователя %s (ID: %d)", userName, chatID)
		if XUIClientInstance != nil {
			if err := XUIClientInstance.Login(); err != nil {
				message := fmt.Sprintf("❌ <b>Ошибка подключения к x-ui:</b>\n%s", err.Error())
				log.Printf("[MessageHandler] Ошибка x-ui для пользователя %s (ID: %d): %v", userName, chatID, err)
				return TelegramBotInstance.SendMessageHTML(chatID, message)
			} else {
				message := "✅ <b>Статус x-ui:</b>\nПодключение успешно установлено"
				log.Printf("[MessageHandler] Статус x-ui успешен для пользователя %s (ID: %d)", userName, chatID)
				return TelegramBotInstance.SendMessageHTML(chatID, message)
			}
		} else {
			message := "❌ <b>Статус x-ui:</b>\nКлиент не инициализирован"
			log.Printf("[MessageHandler] x-ui клиент не инициализирован для пользователя %s (ID: %d)", userName, chatID)
			return TelegramBotInstance.SendMessageHTML(chatID, message)
		}

	case text == "/users" || text == "users":
		log.Printf("[MessageHandler] Выполняется команда /users для пользователя %s (ID: %d)", userName, chatID)
		if UserServiceInstance != nil {
			count, err := UserServiceInstance.GetUsersCount()
			if err != nil {
				log.Printf("[MessageHandler] Ошибка получения количества пользователей: %v", err)
				return TelegramBotInstance.SendMessage(chatID, "Ошибка получения статистики пользователей.")
			}

			message := fmt.Sprintf("📊 <b>Статистика пользователей:</b>\n\nВсего пользователей: %d", count)
			log.Printf("[MessageHandler] Статистика пользователей отправлена для пользователя %s (ID: %d): %d пользователей", userName, chatID, count)
			return TelegramBotInstance.SendMessageHTML(chatID, message)
		} else {
			log.Printf("[MessageHandler] Сервис пользователей не инициализирован для пользователя %s (ID: %d)", userName, chatID)
			return TelegramBotInstance.SendMessage(chatID, "Сервис пользователей не инициализирован.")
		}

	default:
		log.Printf("[MessageHandler] Неизвестная команда от пользователя %s (ID: %d): %s", userName, chatID, update.Message.Text)
		message := fmt.Sprintf("Неизвестная команда: %s\n\nИспользуйте /help для получения справки", update.Message.Text)
		return TelegramBotInstance.SendMessage(chatID, message)
	}
}

func newXUIClient() *xui_client.Client {
	return xui_client.NewClient("http://37.46.19.85:25567/vLr9dnLbg0B140e", "MaKrotos", "3483hiT7")
}

func main() {
	dsn := getDSN()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к Postgres: %v", err)
	}
	defer db.Close()

	if err := goose.Up(db, "internal/migrations"); err != nil {
		log.Fatalf("Ошибка применения миграций: %v", err)
	}

	// Инициализация сервиса пользователей
	UserServiceInstance = telegram.NewUserService(db)
	log.Println("Сервис пользователей инициализирован")

	// Инициализация x-ui клиента
	XUIClientInstance = newXUIClient()
	if err := XUIClientInstance.Login(); err != nil {
		log.Printf("Предупреждение: не удалось подключиться к x-ui: %v", err)
	} else {
		log.Println("Успешное подключение к x-ui")
	}

	// Инициализация Telegram бота
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramToken != "" && telegramToken != "your_bot_token_here" {
		// Определяем режим работы
		mode := telegram.ModePolling // по умолчанию polling
		if envMode := os.Getenv("TELEGRAM_BOT_MODE"); envMode != "" {
			if envMode == "webhook" {
				mode = telegram.ModeWebhook
			}
		}

		// Конфигурация бота
		config := telegram.BotConfig{
			Token:      telegramToken,
			Mode:       mode,
			WebhookURL: os.Getenv("TELEGRAM_WEBHOOK_URL"),
		}

		// Создаем бота
		bot, err := telegram.NewBot(config)
		if err != nil {
			log.Fatalf("Ошибка создания Telegram бота: %v", err)
		}

		// Добавляем обработчик сообщений
		bot.AddHandler(handleTelegramMessage)

		// Сохраняем глобальную ссылку
		TelegramBotInstance = bot

		// Запускаем бота
		if err := bot.Start(); err != nil {
			log.Fatalf("Ошибка запуска Telegram бота: %v", err)
		}

		log.Printf("Telegram бот запущен в режиме: %s", bot.GetMode())
		log.Printf("WebApp URL: %s", WebAppURL)
	} else {
		log.Println("TELEGRAM_BOT_TOKEN не задан или равен значению по умолчанию, Telegram бот не запущен")
	}

	http.HandleFunc("/v1/getUsers", getUsersHandler(db))
	http.HandleFunc("/v1/telegram/users", getTelegramUsersHandler(db))
	log.Println("Сервер запущен на :25566")
	log.Fatal(http.ListenAndServe(":25566", nil))
}

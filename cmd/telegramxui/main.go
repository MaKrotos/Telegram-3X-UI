package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

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
	return "https://37.46.19.85:5173/" // значение по умолчанию (HTTPS)
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

// createVPNForUser создает inbound с пользователем для указанного Telegram пользователя
func createVPNForUser(telegramUserID int64, userName string) (map[string]interface{}, error) {
	if XUIClientInstance == nil {
		return nil, fmt.Errorf("x-ui клиент не инициализирован")
	}

	// Логируем в x-ui
	if err := XUIClientInstance.Login(); err != nil {
		return nil, fmt.Errorf("ошибка входа в x-ui: %w", err)
	}

	// Используем случайный порт для inbound
	port := 20000 + rand.Intn(40000)
	inboundName := fmt.Sprintf("VPN для %s (ID: %d)", userName, telegramUserID)

	log.Printf("[VPN] Создание inbound для пользователя %s (ID: %d) на порту %d", userName, telegramUserID, port)

	emptyInbound := xui_client.GenerateEmptyInboundForm(port, inboundName)
	inboundId, err := XUIClientInstance.AddInbound(emptyInbound)
	if err != nil || inboundId == 0 {
		return nil, fmt.Errorf("ошибка создания inbound: %w", err)
	}

	log.Printf("[VPN] Inbound создан успешно: id=%d, port=%d", inboundId, port)

	// Создаем случайного клиента
	clientId, email, subId, settings := xui_client.GenerateRandomClientSettings(10)
	addClientForm := &xui_client.AddClientForm{
		Id:       inboundId,
		Settings: settings,
	}

	if err := XUIClientInstance.AddClientToInbound(addClientForm); err != nil {
		return nil, fmt.Errorf("ошибка добавления клиента: %w", err)
	}

	log.Printf("[VPN] Клиент добавлен успешно: id=%d, email=%s, subId=%s", clientId, email, subId)

	// Формируем данные для подключения
	connectionData := map[string]interface{}{
		"inbound_id":  inboundId,
		"client_id":   clientId,
		"email":       email,
		"sub_id":      subId,
		"port":        port,
		"settings":    settings,
		"user_name":   userName,
		"telegram_id": telegramUserID,
	}

	return connectionData, nil
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
			message := fmt.Sprintf("Привет, %s! 👋\n\nЯ бот для управления VPN через x-ui.\n\nДоступные команды:\n/status - Статус x-ui\n/create_vpn - Создать VPN подключение\n/help - Помощь", userName)
			log.Printf("[MessageHandler] Отправка fallback сообщения для пользователя %s (ID: %d)", userName, chatID)
			return TelegramBotInstance.SendMessageHTML(chatID, message)
		}
		log.Printf("[MessageHandler] Приветственное сообщение с WebApp отправлено успешно для пользователя %s (ID: %d)", userName, chatID)
		return nil

	case text == "/create_vpn" || text == "create_vpn":
		log.Printf("[MessageHandler] Выполняется команда /create_vpn для пользователя %s (ID: %d)", userName, chatID)

		// Создаем VPN для пользователя
		connectionData, err := createVPNForUser(int64(chatID), userName)
		if err != nil {
			log.Printf("[MessageHandler] Ошибка создания VPN для пользователя %s (ID: %d): %v", userName, chatID, err)
			errorMessage := fmt.Sprintf("❌ <b>Ошибка создания VPN:</b>\n%s", err.Error())
			return TelegramBotInstance.SendMessageHTML(chatID, errorMessage)
		}

		// Формируем сообщение с данными подключения
		message := fmt.Sprintf(`✅ <b>VPN подключение создано!</b>

👤 <b>Пользователь:</b> %s
🆔 <b>ID:</b> %d
📧 <b>Email:</b> %s
🔌 <b>Порт:</b> %d
🆔 <b>Sub ID:</b> %s

<i>Данные для подключения отправлены в следующем сообщении.</i>`,
			userName, chatID, connectionData["email"], connectionData["port"], connectionData["sub_id"])

		log.Printf("[MessageHandler] VPN создан успешно для пользователя %s (ID: %d)", userName, chatID)

		// Отправляем основное сообщение
		if err := TelegramBotInstance.SendMessageHTML(chatID, message); err != nil {
			log.Printf("[MessageHandler] Ошибка отправки основного сообщения: %v", err)
			return err
		}

		// Отправляем данные для подключения в отдельном сообщении
		connectionMessage := fmt.Sprintf(`🔗 <b>Данные для подключения:</b>

🌐 <b>Сервер:</b> 37.46.19.85
🔌 <b>Порт:</b> %d
🔑 <b>UUID:</b> %s
📧 <b>Email:</b> %s
🆔 <b>Sub ID:</b> %s

<i>Используйте эти данные для настройки VPN клиента.</i>`,
			connectionData["port"], connectionData["settings"], connectionData["email"], connectionData["sub_id"])

		log.Printf("[MessageHandler] Отправка данных подключения для пользователя %s (ID: %d)", userName, chatID)
		return TelegramBotInstance.SendMessageHTML(chatID, connectionMessage)

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
/create_vpn - Создать VPN подключение
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
	// Инициализация генератора случайных чисел
	rand.Seed(time.Now().UnixNano())

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

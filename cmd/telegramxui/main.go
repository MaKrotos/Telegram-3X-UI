package main

import (
	"database/sql"
	"log"
	"math/rand"
	"net/http"
	"time"

	"TelegramXUI/internal/config"
	"TelegramXUI/internal/handlers"
	"TelegramXUI/internal/services"
	"TelegramXUI/internal/telegram"
	"TelegramXUI/internal/xui_client"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

func main() {
	// Инициализация генератора случайных чисел
	rand.Seed(time.Now().UnixNano())

	// Загружаем конфигурацию
	cfg := config.Load()

	// Проверяем обязательные переменные окружения
	if cfg.Telegram.Token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не задан в переменных окружения")
	}
	if cfg.XUI.URL == "" {
		log.Fatal("XUI_URL не задан в переменных окружения")
	}
	if cfg.XUI.Username == "" {
		log.Fatal("XUI_USER не задан в переменных окружения")
	}
	if cfg.XUI.Password == "" {
		log.Fatal("XUI_PASSWORD не задан в переменных окружения")
	}
	if cfg.VPN.ServerIP == "" {
		log.Fatal("VPN_SERVER_IP не задан в переменных окружения")
	}

	// Подключаемся к базе данных
	db, err := sql.Open("postgres", cfg.Database.DSN)
	if err != nil {
		log.Fatalf("Ошибка подключения к Postgres: %v", err)
	}
	defer db.Close()

	// Применяем миграции
	if err := goose.Up(db, "internal/migrations"); err != nil {
		log.Fatalf("Ошибка применения миграций: %v", err)
	}

	// Инициализируем сервисы
	userService := services.NewUserService(db)
	log.Println("Сервис пользователей инициализирован")

	// Инициализируем x-ui клиент
	xuiClient := xui_client.NewClient(cfg.XUI.URL, cfg.XUI.Username, cfg.XUI.Password)
	vpnService := services.NewVPNService(xuiClient)

	if err := vpnService.CheckStatus(); err != nil {
		log.Printf("Предупреждение: не удалось подключиться к x-ui: %v", err)
	} else {
		log.Println("Успешное подключение к x-ui")
	}

	// Инициализируем HTTP обработчики
	httpHandler := handlers.NewHTTPHandler(userService, vpnService, cfg.WebApp.URL)

	// Инициализируем Telegram бота
	if cfg.Telegram.Token != "" && cfg.Telegram.Token != "your_bot_token_here" {
		// Определяем режим работы
		mode := telegram.ModePolling // по умолчанию polling
		if cfg.Telegram.Mode == "webhook" {
			mode = telegram.ModeWebhook
		}

		// Конфигурация бота
		botConfig := telegram.BotConfig{
			Token:      cfg.Telegram.Token,
			Mode:       mode,
			WebhookURL: cfg.Telegram.WebhookURL,
		}

		// Создаем бота
		bot, err := telegram.NewBot(botConfig)
		if err != nil {
			log.Fatalf("Ошибка создания Telegram бота: %v", err)
		}

		// Создаем обработчик Telegram сообщений
		telegramHandler := handlers.NewTelegramHandler(vpnService, userService, bot, cfg.WebApp.URL, &cfg.VPN)

		// Добавляем обработчик сообщений
		bot.AddHandler(telegramHandler.HandleMessage)

		// Запускаем бота
		if err := bot.Start(); err != nil {
			log.Fatalf("Ошибка запуска Telegram бота: %v", err)
		}

		log.Printf("Telegram бот запущен в режиме: %s", bot.GetMode())
		log.Printf("WebApp URL: %s", cfg.WebApp.URL)
		log.Printf("VPN Server IP: %s", cfg.VPN.ServerIP)
		log.Printf("VPN Port Range: %d-%d", cfg.VPN.PortRangeStart, cfg.VPN.PortRangeEnd)
	} else {
		log.Println("TELEGRAM_BOT_TOKEN не задан или равен значению по умолчанию, Telegram бот не запущен")
	}

	// Настраиваем HTTP маршруты
	http.HandleFunc("/v1/getUsers", httpHandler.GetUsersHandler())
	http.HandleFunc("/v1/telegram/users", httpHandler.GetTelegramUsersHandler())

	log.Println("Сервер запущен на :25566")
	log.Fatal(http.ListenAndServe(":25566", nil))
}

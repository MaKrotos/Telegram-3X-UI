package main

import (
	"database/sql"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"TelegramXUI/internal/config"
	"TelegramXUI/internal/contracts"
	"TelegramXUI/internal/handlers"
	"TelegramXUI/internal/services"
	"TelegramXUI/internal/telegram"

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

	// Инициализируем сервисы состояний и администратора
	userStateService := services.NewUserStateService(db)
	extensibleStateService := services.NewExtensibleStateService(db)
	xuiServerService := services.NewXUIServerService(db)
	adminService := services.NewAdminService(cfg)

	vpnConnectionService := services.NewVPNConnectionService(db)

	// Создаем сервис для добавления XUI хостов
	xuiHostAddService := services.NewXUIHostAddService(
		userStateService,
		extensibleStateService,
		xuiServerService,
		adminService,
	)

	// Инициализируем HTTP обработчики
	httpHandler := handlers.NewHTTPHandler(userService, nil, cfg.WebApp.URL)

	// Переменные для graceful shutdown
	var bot *telegram.TelegramBot
	var hostMonitorService *services.HostMonitorService

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
		bot, err = telegram.NewBot(botConfig)
		if err != nil {
			log.Fatalf("Ошибка создания Telegram бота: %v", err)
		}

		// Создаем адаптеры для совместимости типов
		userStateAdapter := &UserStateServiceAdapter{userStateService}
		xuiHostAddAdapter := &XUIHostAddServiceAdapter{xuiHostAddService}

		// Создаем адаптер для TelegramClient
		telegramClientAdapter := &TelegramClientAdapter{bot}

		// Создаем сервис мониторинга хостов
		hostMonitorService = services.NewHostMonitorService(
			xuiServerService,
			adminService,
			telegramClientAdapter,
			time.Duration(cfg.Monitor.CheckIntervalMinutes)*time.Minute,
		)

		// Создаем Telegram UserService
		telegramUserService := telegram.NewUserService(db)

		// Создаем новый обработчик сообщений с поддержкой добавления XUI хостов и мониторинга
		messageProcessor := telegram.NewMessageProcessor(
			userStateAdapter,
			extensibleStateService,
			xuiHostAddAdapter,
			adminService,
			xuiServerService,
			hostMonitorService,
			telegramUserService,
			vpnConnectionService,
			cfg,
		)

		// Добавляем обработчик сообщений
		bot.AddHandler(messageProcessor.ProcessMessage)

		// Запускаем мониторинг хостов
		if err := hostMonitorService.Start(); err != nil {
			log.Printf("Предупреждение: не удалось запустить мониторинг хостов: %v", err)
		} else {
			log.Printf("Мониторинг хостов запущен с интервалом %d минут", cfg.Monitor.CheckIntervalMinutes)
		}

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

	// Создаем канал для обработки сигналов
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Запускаем сервер в отдельной горутине
	go func() {
		if err := http.ListenAndServe(":25566", nil); err != nil {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	// Ожидаем сигнала для завершения
	<-signalChan

	log.Println("Сервер завершает работу")

	// Graceful shutdown
	if hostMonitorService != nil {
		log.Println("Останавливаем мониторинг хостов...")
		if err := hostMonitorService.Stop(); err != nil {
			log.Printf("Ошибка остановки мониторинга: %v", err)
		}
	}

	if bot != nil {
		log.Println("Останавливаем Telegram бота...")
		if err := bot.Stop(); err != nil {
			log.Printf("Ошибка остановки бота: %v", err)
		}
	}

	log.Println("Сервер успешно завершил работу")
}

// UserStateServiceAdapter адаптирует services.UserStateService к contracts.UserStateService
type UserStateServiceAdapter struct {
	service *services.UserStateService
}

func (a *UserStateServiceAdapter) GetUserState(telegramID int64) (*contracts.UserStateInfo, error) {
	user, err := a.service.GetUserState(telegramID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	// Преобразуем services.UserStateInfo в contracts.UserStateInfo
	return &contracts.UserStateInfo{
		ID:                     user.ID,
		TelegramID:             user.TelegramID,
		Username:               user.Username,
		FirstName:              user.FirstName,
		LastName:               user.LastName,
		State:                  string(user.State),
		ExpectedAction:         string(user.ExpectedAction),
		StateChangedAt:         user.StateChangedAt,
		StateReason:            user.StateReason,
		StateChangedByTgID:     user.StateChangedByTgID,
		StateChangedByUsername: user.StateChangedByUsername,
		StateExpiresAt:         user.StateExpiresAt,
		StateMetadata:          user.StateMetadata,
		CreatedAt:              user.CreatedAt,
		UpdatedAt:              user.UpdatedAt,
		LastActivity:           user.LastActivity,
	}, nil
}

func (a *UserStateServiceAdapter) CanUserPerformAction(telegramID int64) (bool, string, error) {
	return a.service.CanUserPerformAction(telegramID)
}

// XUIHostAddServiceAdapter адаптирует services.XUIHostAddService к contracts.XUIHostAddService
type XUIHostAddServiceAdapter struct {
	service *services.XUIHostAddService
}

func (a *XUIHostAddServiceAdapter) StartAddHostProcess(telegramID int64, username string) error {
	return a.service.StartAddHostProcess(telegramID, username)
}

func (a *XUIHostAddServiceAdapter) ProcessHostData(telegramID int64, message string, username string) (*contracts.XUIHostData, error) {
	hostData, err := a.service.ProcessHostData(telegramID, message, username)
	if err != nil {
		return nil, err
	}

	// Преобразуем services.XUIHostData в contracts.XUIHostData
	return &contracts.XUIHostData{
		Host:      hostData.Host,
		Login:     hostData.Login,
		Password:  hostData.Password,
		SecretKey: hostData.SecretKey,
	}, nil
}

func (a *XUIHostAddServiceAdapter) CancelAddHostProcess(telegramID int64, username string) error {
	return a.service.CancelAddHostProcess(telegramID, username)
}

func (a *XUIHostAddServiceAdapter) GetAddHostInstructions() string {
	return a.service.GetAddHostInstructions()
}

func (a *XUIHostAddServiceAdapter) IsInAddHostState(telegramID int64) (bool, error) {
	return a.service.IsInAddHostState(telegramID)
}

// TelegramClientAdapter адаптирует telegram.TelegramBot к интерфейсу contracts.TelegramMessageSender
type TelegramClientAdapter struct {
	bot *telegram.TelegramBot
}

func (a *TelegramClientAdapter) SendMessage(chatID int64, message string) error {
	return a.bot.SendMessage(int(chatID), message)
}

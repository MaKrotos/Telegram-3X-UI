package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config содержит конфигурацию приложения
type Config struct {
	Database DatabaseConfig
	XUI      XUIConfig
	Telegram TelegramConfig
	WebApp   WebAppConfig
	VPN      VPNConfig
	Admin    AdminConfig
	Monitor  MonitorConfig
}

// DatabaseConfig содержит конфигурацию базы данных
type DatabaseConfig struct {
	DSN string
}

// XUIConfig содержит конфигурацию x-ui
type XUIConfig struct {
	URL      string
	Username string
	Password string
}

// TelegramConfig содержит конфигурацию Telegram бота
type TelegramConfig struct {
	Token       string
	Mode        string
	WebhookURL  string
	WebhookPort string
}

// WebAppConfig содержит конфигурацию WebApp
type WebAppConfig struct {
	URL string
}

// VPNConfig содержит конфигурацию VPN сервера
type VPNConfig struct {
	ServerIP       string
	PortRangeStart int
	PortRangeEnd   int
}

// AdminConfig содержит конфигурацию администратора
type AdminConfig struct {
	GlobalAdminTgID     int64
	GlobalAdminUsername string
}

// MonitorConfig содержит конфигурацию мониторинга хостов
type MonitorConfig struct {
	CheckIntervalMinutes int
}

// Load загружает конфигурацию из переменных окружения
func Load() *Config {
	return &Config{
		Database: DatabaseConfig{
			DSN: getDSN(),
		},
		XUI: XUIConfig{
			URL:      getEnvOrDefault("XUI_URL", ""),
			Username: getEnvOrDefault("XUI_USER", ""),
			Password: getEnvOrDefault("XUI_PASSWORD", ""),
		},
		Telegram: TelegramConfig{
			Token:       getEnvOrDefault("TELEGRAM_BOT_TOKEN", ""),
			Mode:        getEnvOrDefault("TELEGRAM_BOT_MODE", "polling"),
			WebhookURL:  getEnvOrDefault("TELEGRAM_WEBHOOK_URL", ""),
			WebhookPort: getEnvOrDefault("TELEGRAM_WEBHOOK_PORT", "8080"),
		},
		WebApp: WebAppConfig{
			URL: getWebAppURL(),
		},
		VPN: VPNConfig{
			ServerIP:       getEnvOrDefault("VPN_SERVER_IP", ""),
			PortRangeStart: getEnvAsInt("VPN_SERVER_PORT_RANGE_START", 20000),
			PortRangeEnd:   getEnvAsInt("VPN_SERVER_PORT_RANGE_END", 60000),
		},
		Admin: AdminConfig{
			GlobalAdminTgID:     getEnvAsInt64("GLOBAL_ADMIN_TG_ID", 0),
			GlobalAdminUsername: getEnvOrDefault("GLOBAL_ADMIN_USERNAME", ""),
		},
		Monitor: MonitorConfig{
			CheckIntervalMinutes: getEnvAsInt("HOST_MONITOR_INTERVAL_MINUTES", 5),
		},
	}
}

// getDSN формирует строку подключения к базе данных
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

// getWebAppURL получает URL для WebApp
func getWebAppURL() string {
	if url := os.Getenv("TELEGRAM_WEBAPP_URL"); url != "" {
		return url
	}
	return "" // значение по умолчанию
}

// getEnvOrDefault получает значение переменной окружения или возвращает значение по умолчанию
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt получает значение переменной окружения как int или возвращает значение по умолчанию
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsInt64 получает значение переменной окружения как int64 или возвращает значение по умолчанию
func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

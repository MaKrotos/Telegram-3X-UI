# TelegramXUI

Telegram-бот для автоматического создания VPN подключений через панель x-ui с поддержкой inline кнопок и хранением пользователей в PostgreSQL.

## 🚀 Возможности

- **Telegram Bot с inline кнопками** - удобное создание VPN через кнопки
- **Автоматическое подключение к x-ui** - управление через REST API
- **Создание VPN подключений** - автоматическое создание inbound и клиентов
- **Хранение пользователей** - база данных PostgreSQL с миграциями
- **REST API** - для получения статистики и пользователей
- **Поддержка Docker** - легкий запуск через docker-compose

## 🏗️ Архитектура

```
TelegramXUI/
├── cmd/telegramxui/          # Точка входа приложения
├── internal/
│   ├── config/              # Конфигурация и переменные окружения
│   ├── handlers/            # HTTP и Telegram обработчики
│   ├── services/            # Бизнес-логика (VPN, пользователи)
│   ├── telegram/            # Telegram API клиент
│   ├── xui_client/          # x-ui API клиент
│   └── migrations/          # Миграции PostgreSQL
├── docker-compose.yml       # Docker конфигурация
├── run-local.ps1           # Скрипт запуска для Windows
└── SETUP.md                # Подробная инструкция по настройке
```

## ⚡ Быстрый старт

### 1. Docker Compose (рекомендуется)

```bash
# Клонируйте репозиторий
git clone https://github.com/yourname/TelegramXUI.git
cd TelegramXUI

# Запустите все сервисы
docker-compose up -d

# Просмотр логов
docker-compose logs -f app
```

### 2. Локальная разработка

#### Windows (PowerShell):
```powershell
.\run-local.ps1
```

#### Linux/Mac:
```bash
# Установите переменные окружения
export TELEGRAM_BOT_TOKEN="your_bot_token"
export XUI_URL="http://your-server:port/path"
export XUI_USER="your_username"
export XUI_PASSWORD="your_password"
export VPN_SERVER_IP="your_server_ip"

# Запустите приложение
go run cmd/telegramxui/main.go
```

## ⚙️ Конфигурация

### Обязательные переменные окружения

Все настройки должны быть заданы в переменных окружения:

**Docker окружение (`docker-compose.yml`):**
```yaml
environment:
  TELEGRAM_BOT_TOKEN: "your_bot_token"
  XUI_URL: "http://your-server:port/path"
  XUI_USER: "your_username"
  XUI_PASSWORD: "your_password"
  VPN_SERVER_IP: "your_server_ip"
  TELEGRAM_WEBAPP_URL: "http://your-webapp-url"
```

**Локальная разработка (`run-local.ps1`):**
```powershell
$env:TELEGRAM_BOT_TOKEN = "your_bot_token"
$env:XUI_URL = "http://your-server:port/path"
$env:XUI_USER = "your_username"
$env:XUI_PASSWORD = "your_password"
$env:VPN_SERVER_IP = "your_server_ip"
$env:TELEGRAM_WEBAPP_URL = "http://your-webapp-url"
```

### Получение токена Telegram бота

1. Найдите @BotFather в Telegram
2. Отправьте команду `/newbot`
3. Следуйте инструкциям
4. Скопируйте полученный токен в конфигурацию

## 🤖 Telegram Bot Команды

- `/start` - Начать работу с ботом (меню с кнопками)
- `/help` - Справка по командам
- `/status` - Проверка статуса x-ui
- `/users` - Статистика пользователей

## 📡 API Endpoints

- `GET /v1/getUsers` - Получение пользователей и статуса x-ui
- `GET /v1/telegram/users` - Получение Telegram пользователей

## 🗄️ База данных

Проект использует PostgreSQL с автоматическими миграциями:

```sql
-- Таблица пользователей Telegram
CREATE TABLE telegram_users (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    first_name TEXT,
    last_name TEXT,
    username TEXT,
    is_bot BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 🔧 Разработка

### Добавление новых команд

1. Добавьте обработку в `internal/handlers/telegram_handler.go`
2. Обновите справку в методе `handleHelpCommand`

### Добавление новых API endpoints

1. Создайте обработчик в `internal/handlers/http_handler.go`
2. Добавьте маршрут в `cmd/telegramxui/main.go`

### Структура сервисов

- `VPNService` - создание VPN подключений через x-ui
- `UserService` - управление пользователями в БД
- `TelegramHandler` - обработка сообщений и callback query
- `HTTPHandler` - REST API endpoints

## 🐛 Troubleshooting

### Бот не отвечает
- Проверьте токен в конфигурации
- Убедитесь, что бот не заблокирован
- Проверьте логи: `docker-compose logs app`

### Ошибка подключения к x-ui
- Проверьте URL, логин и пароль
- Убедитесь, что x-ui доступен
- Проверьте права пользователя

### Ошибка базы данных
- Убедитесь, что PostgreSQL запущен
- Проверьте параметры подключения
- Примените миграции: `goose up`

## 🔒 Безопасность

- Никогда не коммитьте токены в код
- Используйте HTTPS для webhook
- Регулярно обновляйте пароли
- Ограничьте доступ к API endpoints

## 📚 Дополнительная документация

Подробная инструкция по настройке и развертыванию: [SETUP.md](SETUP.md)

## 📄 Лицензия

MIT

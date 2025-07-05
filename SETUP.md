# TelegramXUI - Настройка и запуск

## Быстрый старт

### 1. Подготовка окружения

Все настройки уже настроены в `docker-compose.yml` и `run-local.ps1`. 

**Для изменения настроек отредактируйте:**
- `docker-compose.yml` - для Docker окружения
- `run-local.ps1` - для локальной разработки на Windows

### 2. Запуск с Docker Compose

```bash
# Запуск всех сервисов
docker-compose up -d

# Просмотр логов
docker-compose logs -f app
```

### 3. Локальная разработка

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

## Конфигурация

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

**Приложение не запустится без этих переменных!**

### Получение токена Telegram бота

1. Найдите @BotFather в Telegram
2. Отправьте команду `/newbot`
3. Следуйте инструкциям
4. Скопируйте полученный токен в `.env`

### Настройка x-ui

1. Убедитесь, что x-ui доступен по указанному URL
2. Проверьте логин и пароль
3. Убедитесь, что у пользователя есть права на создание inbound

## Архитектура

```
TelegramXUI/
├── cmd/telegramxui/          # Точка входа
├── internal/
│   ├── config/              # Конфигурация
│   ├── handlers/            # HTTP и Telegram обработчики
│   ├── services/            # Бизнес-логика
│   ├── telegram/            # Telegram API клиент
│   ├── xui_client/          # x-ui API клиент
│   └── migrations/          # Миграции БД
├── docker-compose.yml       # Docker конфигурация
├── run-local.ps1           # Скрипт запуска для Windows
└── env.example             # Пример конфигурации
```

## API Endpoints

- `GET /v1/getUsers` - Получение пользователей и статуса x-ui
- `GET /v1/telegram/users` - Получение Telegram пользователей

## Telegram Bot Команды

- `/start` - Начать работу с ботом (меню с кнопками)
- `/help` - Справка по командам
- `/status` - Проверка статуса x-ui
- `/users` - Статистика пользователей

## Разработка

### Добавление новых команд

1. Добавьте обработку в `internal/handlers/telegram_handler.go`
2. Обновите справку в методе `handleHelpCommand`

### Добавление новых API endpoints

1. Создайте обработчик в `internal/handlers/http_handler.go`
2. Добавьте маршрут в `cmd/telegramxui/main.go`

## Troubleshooting

### Бот не отвечает
- Проверьте токен в `.env`
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

## Безопасность

- Никогда не коммитьте `.env` файл
- Используйте HTTPS для webhook
- Регулярно обновляйте пароли
- Ограничьте доступ к API endpoints 
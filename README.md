# TelegramXUI

Telegram-бот для автоматического создания VPN подключений через панель x-ui с поддержкой inline кнопок и хранением пользователей в PostgreSQL.

## 🚀 Возможности

- **Telegram Bot с inline кнопками** - удобное создание VPN через кнопки
- **Автоматическое подключение к x-ui** - управление через REST API
- **Создание VPN подключений** - автоматическое создание inbound и клиентов
- **Мониторинг хостов** - автоматическая проверка доступности XUI хостов
- **Уведомления администраторов** - оповещения о неактивных хостах
- **Управление множественными хостами** - добавление и мониторинг нескольких серверов
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
export VPN_SERVER_IP="your_server_ip"
export GLOBAL_ADMIN_TG_ID="your_telegram_id"
export GLOBAL_ADMIN_USERNAME="your_username"

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
  VPN_SERVER_IP: "your_server_ip"
  TELEGRAM_WEBAPP_URL: "http://your-webapp-url"
  GLOBAL_ADMIN_TG_ID: "your_telegram_id"
  GLOBAL_ADMIN_USERNAME: "your_username"
  HOST_MONITOR_INTERVAL_MINUTES: "5"
```

**Локальная разработка (`run-local.ps1`):**
```powershell
$env:TELEGRAM_BOT_TOKEN = "your_bot_token"
$env:VPN_SERVER_IP = "your_server_ip"
$env:TELEGRAM_WEBAPP_URL = "http://your-webapp-url"
$env:GLOBAL_ADMIN_TG_ID = "your_telegram_id"
$env:GLOBAL_ADMIN_USERNAME = "your_username"
$env:HOST_MONITOR_INTERVAL_MINUTES = "5"
```

### Получение токена Telegram бота

1. Найдите @BotFather в Telegram
2. Отправьте команду `/newbot`
3. Следуйте инструкциям
4. Скопируйте полученный токен в конфигурацию

### Настройка глобального администратора

1. **Получите ваш Telegram ID**:
   - Отправьте сообщение боту @userinfobot
   - Скопируйте ваш ID

2. **Укажите username**:
   - Ваш username в Telegram (без @)

3. **Добавьте в конфигурацию**:
   ```bash
   export GLOBAL_ADMIN_TG_ID="ваш_id"
   export GLOBAL_ADMIN_USERNAME="ваш_username"
   ```

## 🤖 Telegram Bot Команды

### Для всех пользователей:
- `/start` - Начать работу с ботом (меню с кнопками)
- `/help` - Справка по командам
- `/cancel` - Отменить текущую операцию
- `/vpn` - Управление VPN подключениями

### Для администраторов:
- `/addhost` - Добавить новый XUI хост
- `/monitor` - Управление мониторингом хостов
- `/monitor_start` - Запустить мониторинг
- `/monitor_stop` - Остановить мониторинг
- `/monitor_status` - Статус мониторинга
- `/check_hosts` - Проверить все хосты сейчас

## 📡 API Endpoints

- `GET /v1/getUsers` - Получение пользователей и статуса x-ui
- `GET /v1/telegram/users` - Получение Telegram пользователей

## 🔍 Мониторинг хостов

Система автоматически мониторит все добавленные XUI хосты:

### Автоматические проверки
- **Интервал проверки**: настраивается через `HOST_MONITOR_INTERVAL_MINUTES` (по умолчанию 5 минут)
- **Проверка авторизации**: попытка входа в панель XUI
- **Проверка API**: тестирование получения списка пользователей
- **Автоматическое отключение**: неактивные хосты помечаются как неактивные

### Уведомления
- **Получатели**: глобальные администраторы (настраиваются через `GLOBAL_ADMIN_TG_ID` и `GLOBAL_ADMIN_USERNAME`)
- **Формат**: подробное сообщение с описанием проблемы и временем проверки
- **Действие**: неактивные хосты автоматически исключаются из пула для создания VPN

### Управление мониторингом
- `/monitor` - общая информация о мониторинге
- `/monitor_start` - запуск автоматических проверок
- `/monitor_stop` - остановка автоматических проверок
- `/monitor_status` - детальная статистика хостов
- `/check_hosts` - немедленная проверка всех хостов

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

-- Таблица XUI серверов
CREATE TABLE xui_servers (
    id SERIAL PRIMARY KEY,
    server_url VARCHAR(255) NOT NULL,
    server_name VARCHAR(255),
    username VARCHAR(255),
    password VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    added_by_tg_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица VPN подключений
CREATE TABLE vpn_connections (
    id SERIAL PRIMARY KEY,
    telegram_user_id BIGINT NOT NULL,
    server_id INTEGER NOT NULL,
    inbound_id INTEGER NOT NULL,
    client_id VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL,
    vless_link TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
- `XUIServerService` - управление XUI серверами
- `HostMonitorService` - мониторинг хостов
- `AdminService` - управление правами администраторов
- `TelegramHandler` - обработка сообщений и callback query
- `HTTPHandler` - REST API endpoints

## 🐛 Troubleshooting

### Бот не отвечает
- Проверьте токен в конфигурации
- Убедитесь, что бот не заблокирован
- Проверьте логи: `docker-compose logs app`

### Ошибка подключения к x-ui
- Добавьте XUI хост через команду `/addhost`
- Проверьте правильность данных хоста
- Убедитесь, что хост доступен

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

# Локальная разработка

## 🚀 Быстрый старт

### Вариант 1: Запуск через Docker (рекомендуется)

```bash
# Запуск всех сервисов
docker compose up

# Или в фоновом режиме
docker compose up -d
```

### Вариант 2: Локальный запуск

#### Предварительные требования:
1. **PostgreSQL** должен быть запущен
2. **Go** установлен
3. **База данных** создана

#### Настройка базы данных:

1. **Запустите PostgreSQL:**
```bash
# Через Docker
docker run -d \
  --name postgres_local \
  -e POSTGRES_DB=telegramxui \
  -e POSTGRES_USER=telegramxui_user \
  -e POSTGRES_PASSWORD=telegramxui_password \
  -p 5432:5432 \
  postgres:16-alpine
```

2. **Или используйте существующий PostgreSQL**

#### Запуск приложения:

**Windows (PowerShell):**
```powershell
.\run-local.ps1
```

**Windows (CMD):**
```cmd
run-local.bat
```

**Linux/Mac:**
```bash
export POSTGRES_DSN="postgres://telegramxui_user:telegramxui_password@localhost:5432/telegramxui?sslmode=disable"
export TELEGRAM_BOT_TOKEN="your_bot_token_here"
export TELEGRAM_BOT_MODE="polling"
export TELEGRAM_WEBAPP_URL="http://another:port"
export VPN_SERVER_IP="http://another:port"
export GLOBAL_ADMIN_TG_ID="your_telegram_id"
export GLOBAL_ADMIN_USERNAME="your_username"
export HOST_MONITOR_INTERVAL_MINUTES="5"

go run cmd/telegramxui/main.go
```

## 🔧 Настройка переменных окружения

### Обязательные переменные:

| Переменная | Описание | Пример |
|------------|----------|--------|
| `POSTGRES_DSN` | Строка подключения к БД | `postgres://user:pass@localhost:5432/db?sslmode=disable` |
| `TELEGRAM_BOT_TOKEN` | Токен бота от @BotFather | `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz` |
| `GLOBAL_ADMIN_TG_ID` | Telegram ID администратора | `123456789` |
| `GLOBAL_ADMIN_USERNAME` | Username администратора | `admin_username` |

### Опциональные переменные:

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `TELEGRAM_BOT_MODE` | Режим бота | `polling` |
| `TELEGRAM_WEBAPP_URL` | URL WebApp | `http://another:port/` |
| `VPN_SERVER_IP` | IP VPN сервера | `126.45.45.45.45` |
| `HOST_MONITOR_INTERVAL_MINUTES` | Интервал мониторинга | `5` |

## 🗄️ Структура базы данных

При первом запуске автоматически создаются таблицы:

```sql
-- Основная таблица пользователей
CREATE TABLE telegram_users (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255),
    is_bot BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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

## 🔍 Отладка

### Проверка подключения к БД:
```bash
# Проверка доступности PostgreSQL
psql -h localhost -U telegramxui_user -d telegramxui -c "SELECT 1;"
```

### Проверка API:
```bash
# Проверка основного API
curl http://localhost:25566/v1/getUsers

# Проверка Telegram пользователей
curl http://localhost:25566/v1/telegram/users
```

### Логи:
```bash
# Docker логи
docker compose logs app

# Локальные логи выводятся в консоль
```

## 🐛 Решение проблем

### Ошибка "password authentication failed":
- Проверьте правильность `POSTGRES_DSN`
- Убедитесь, что PostgreSQL запущен
- Проверьте пользователя и пароль

### Ошибка "connection refused":
- PostgreSQL не запущен
- Неправильный порт (по умолчанию 5432)

### Ошибка "database does not exist":
- Создайте базу данных: `CREATE DATABASE telegramxui;`

### Telegram бот не отвечает:
- Проверьте `TELEGRAM_BOT_TOKEN`
- Убедитесь, что токен правильный
- Проверьте логи на ошибки

### Нет XUI хостов:
- Добавьте хосты через команду `/addhost` в боте
- Убедитесь, что вы являетесь глобальным администратором
- Проверьте правильность данных хоста

## 📁 Структура проекта

```
Telegram-3X-UI/
├── cmd/telegramxui/
│   └── main.go              # Точка входа
├── internal/
│   ├── telegram/
│   │   ├── client.go        # Telegram API клиент
│   │   ├── bot.go           # Универсальный бот
│   │   └── user_service.go  # Сервис пользователей
│   ├── xui_client/
│   │   └── client.go        # X-UI API клиент
│   └── migrations/
│       └── *.sql            # Миграции БД
├── docker-compose.yml       # Docker конфигурация
├── run-local.bat           # Скрипт запуска (Windows CMD)
├── run-local.ps1           # Скрипт запуска (PowerShell)
└── LOCAL_DEVELOPMENT.md    # Эта инструкция
```

## 🔄 Разработка

### Добавление новых команд бота:
1. Отредактируйте `handleTelegramMessage` в `main.go`
2. Добавьте новый case в switch
3. Перезапустите приложение

### Добавление новых API endpoints:
1. Создайте новый handler в `main.go`
2. Добавьте маршрут в `main()`
3. Перезапустите приложение

### Изменение структуры БД:
1. Создайте новую миграцию в `internal/migrations/`
2. Перезапустите приложение (миграции применятся автоматически)

### Добавление XUI хостов:
1. Запустите бота
2. Отправьте команду `/addhost`
3. Введите данные хоста в формате: `хост логин пароль [секретный_ключ]`
4. Система автоматически проверит подключение и добавит хост 
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
export XUI_URL="http://37.46.19.85:25567/vLr9dnLbg0B140e/"
export XUI_USER="MaKrotos"
export XUI_PASSWORD="3483hiT7"
export TELEGRAM_BOT_TOKEN="your_bot_token_here"
export TELEGRAM_BOT_MODE="polling"
export TELEGRAM_WEBAPP_URL="http://37.46.19.85:5173/"

go run cmd/telegramxui/main.go
```

## 🔧 Настройка переменных окружения

### Обязательные переменные:

| Переменная | Описание | Пример |
|------------|----------|--------|
| `POSTGRES_DSN` | Строка подключения к БД | `postgres://user:pass@localhost:5432/db?sslmode=disable` |
| `TELEGRAM_BOT_TOKEN` | Токен бота от @BotFather | `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz` |

### Опциональные переменные:

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `XUI_URL` | URL x-ui панели | `http://37.46.19.85:25567/vLr9dnLbg0B140e/` |
| `XUI_USER` | Логин x-ui | `MaKrotos` |
| `XUI_PASSWORD` | Пароль x-ui | `3483hiT7` |
| `TELEGRAM_BOT_MODE` | Режим бота | `polling` |
| `TELEGRAM_WEBAPP_URL` | URL WebApp | `http://37.46.19.85:5173/` |

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
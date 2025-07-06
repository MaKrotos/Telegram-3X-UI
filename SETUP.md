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
export VPN_SERVER_IP="your_server_ip"
export GLOBAL_ADMIN_TG_ID="your_telegram_id"
export GLOBAL_ADMIN_USERNAME="your_username"

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

**Приложение не запустится без этих переменных!**

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

### Добавление XUI хостов

После запуска системы администраторы могут добавлять XUI хосты через бота:

1. Отправьте команду `/addhost`
2. Введите данные в формате: `хост логин пароль [секретный_ключ]`
3. Система автоматически проверит подключение
4. Хост будет добавлен в базу данных

**Пример:**
```
https://xui.example.com admin password123
```

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
- `/vpn` - Управление VPN подключениями
- `/addhost` - Добавить XUI хост (только для администраторов)
- `/monitor` - Управление мониторингом хостов (только для администраторов)

## Разработка

### Добавление новых команд

1. Добавьте обработку в `internal/handlers/telegram_handler.go`
2. Обновите справку в методе `handleHelpCommand`

### Добавление новых API endpoints

1. Создайте обработчик в `internal/handlers/http_handler.go`
2. Добавьте маршрут в `cmd/telegramxui/main.go`

## Troubleshooting

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

## Безопасность

- Никогда не коммитьте токены в код
- Используйте HTTPS для webhook
- Регулярно обновляйте пароли
- Ограничьте доступ к API endpoints 
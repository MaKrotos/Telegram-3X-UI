# Настройка Telegram Bot

## 🚀 Быстрый старт

### 1. Получение токена бота

1. Найдите [@BotFather](https://t.me/BotFather) в Telegram
2. Отправьте команду `/newbot`
3. Следуйте инструкциям:
   - Введите имя бота (например: "My VPN Bot")
   - Введите username бота (например: "my_vpn_bot")
4. Скопируйте полученный токен (выглядит как `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz`)

### 2. Настройка для разработки (Polling режим)

Создайте файл `docker-compose.yml`:

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: postgres_local
    environment:
      POSTGRES_DB: telegramxui
      POSTGRES_USER: fakeuser
      POSTGRES_PASSWORD: fakepassword
    ports:
      - "5432:5432"
    volumes:
      - ./pg_data:/var/lib/postgresql/data

  app:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: telegramxui_app
    depends_on:
      - postgres
    environment:
      POSTGRES_DSN: "postgres://fakeuser:fakepassword@postgres:5432/telegramxui?sslmode=disable"
      XUI_URL: "http://your-xui-server:25567/your-path/"
      XUI_USER: "your_xui_username"
      XUI_PASSWORD: "your_xui_password"
      
      # Telegram Bot настройки
      TELEGRAM_BOT_TOKEN: "1234567890:ABCdefGHIjklMNOpqrsTUVwxyz"
      TELEGRAM_BOT_MODE: "polling"
      
    command: ["air"]
    ports:
      - "25566:25566"
    volumes:
      - ./:/app
```

### 3. Запуск

```bash
docker compose up
```

## 🌐 Настройка для продакшена (Webhook режим)

### 1. Подготовка домена

Вам нужен:
- Домен с SSL сертификатом
- Публичный IP или домен
- Настроенный reverse proxy (nginx/traefik)

### 2. Настройка nginx

Создайте конфигурацию nginx:

```nginx
server {
    listen 443 ssl;
    server_name your-domain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location /webhook {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 3. Docker Compose для продакшена

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: postgres_local
    environment:
      POSTGRES_DB: telegramxui
      POSTGRES_USER: fakeuser
      POSTGRES_PASSWORD: fakepassword
    ports:
      - "5432:5432"
    volumes:
      - ./pg_data:/var/lib/postgresql/data

  app:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: telegramxui_app
    depends_on:
      - postgres
    environment:
      POSTGRES_DSN: "postgres://fakeuser:fakepassword@postgres:5432/telegramxui?sslmode=disable"
      XUI_URL: "http://your-xui-server:25567/your-path/"
      XUI_USER: "your_xui_username"
      XUI_PASSWORD: "your_xui_password"
      
      # Telegram Bot настройки
      TELEGRAM_BOT_TOKEN: "1234567890:ABCdefGHIjklMNOpqrsTUVwxyz"
      TELEGRAM_BOT_MODE: "webhook"
      TELEGRAM_WEBHOOK_URL: "https://your-domain.com/webhook"
      TELEGRAM_WEBHOOK_PORT: "8080"
      
    command: ["air"]
    ports:
      - "25566:25566"
      - "8080:8080"
    volumes:
      - ./:/app
```

## 📱 Доступные команды бота

После запуска бота вы можете использовать следующие команды:

- `/start` - Начать работу с ботом
- `/help` - Показать справку
- `/status` - Проверить статус x-ui

## 🔧 Переменные окружения

| Переменная | Описание | Обязательная | По умолчанию |
|------------|----------|--------------|--------------|
| `TELEGRAM_BOT_TOKEN` | Токен бота от @BotFather | ✅ | - |
| `TELEGRAM_BOT_MODE` | Режим работы: `polling` или `webhook` | ❌ | `polling` |
| `TELEGRAM_WEBHOOK_URL` | URL для webhook (только для webhook режима) | ❌ | - |
| `TELEGRAM_WEBHOOK_PORT` | Порт для webhook сервера | ❌ | `8080` |

## 🔍 Отладка

### Проверка статуса бота

```bash
# Проверка логов
docker compose logs app

# Проверка переменных окружения
docker compose exec app env | grep TELEGRAM
```

### Тестирование webhook

```bash
# Проверка webhook информации
curl -X POST http://localhost:25566/v1/webhook-info
```

## 🚨 Безопасность

1. **Никогда не коммитьте токены в git**
   - Файл `docker-compose.yml` уже добавлен в `.gitignore`
   - Используйте переменные окружения или секреты

2. **Используйте HTTPS для webhook**
   - Telegram требует HTTPS для webhook
   - Используйте Let's Encrypt для бесплатных сертификатов

3. **Ограничьте доступ к webhook**
   - Настройте firewall
   - Используйте аутентификацию если нужно

## 📞 Поддержка

Если возникли проблемы:

1. Проверьте логи: `docker compose logs app`
2. Убедитесь, что токен правильный
3. Для webhook проверьте доступность HTTPS URL
4. Проверьте настройки nginx/reverse proxy

## 🔄 Переключение режимов

### С Polling на Webhook:

1. Настройте домен и SSL
2. Измените `TELEGRAM_BOT_MODE` на `webhook`
3. Добавьте `TELEGRAM_WEBHOOK_URL`
4. Перезапустите контейнеры

### С Webhook на Polling:

1. Измените `TELEGRAM_BOT_MODE` на `polling`
2. Удалите `TELEGRAM_WEBHOOK_URL`
3. Перезапустите контейнеры 
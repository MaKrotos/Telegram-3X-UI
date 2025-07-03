# TelegramXUI

Телеграм-бот и сервис для автоматизации работы с панелью 3x-ui (X-UI) через REST API и хранения пользователей в PostgreSQL.

## Возможности

- Автоматическое подключение к панели 3x-ui
- Добавление и управление inbound и клиентами через API
- Хранение пользователей в базе данных PostgreSQL
- REST API для получения пользователей

## Быстрый старт (Docker Compose)

1. **Клонируйте репозиторий:**

   ```bash
   git clone https://github.com/yourname/TelegramXUI.git
   cd TelegramXUI
   ```

2. **Настройте переменные окружения** (по умолчанию уже заданы в `docker-compose.yml`):

   - `POSTGRES_DSN` — строка подключения к PostgreSQL
   - `XUI_URL` — адрес панели 3x-ui (например, http://127.0.0.1:54321)
   - `XUI_USER` — логин для панели 3x-ui
   - `XUI_PASSWORD` — пароль для панели 3x-ui

   Если нужно, измените их в файле `docker-compose.yml`:

   ```yaml
   environment:
     POSTGRES_DSN: "postgres://user:password@postgres:5432/telegramxui?sslmode=disable"
     XUI_URL: "http://127.0.0.1:54321"
     XUI_USER: "admin"
     XUI_PASSWORD: "admin"
   ```

3. **Запустите сервисы:**

   ```bash
   docker-compose up --build
   ```

4. **API будет доступен на порту 25566**
   - Пример запроса: [http://localhost:25566/v1/getUsers](http://localhost:25566/v1/getUsers)

## Структура проекта

- `cmd/telegramxui/main.go` — основной сервер и логика работы
- `internal/xui_client/` — Go-клиент для работы с 3x-ui API
- `internal/migrations/` — миграции для PostgreSQL
- `docker-compose.yml` — запуск через Docker

## Миграции и структура БД

- Таблица пользователей:
  ```sql
  CREATE TABLE IF NOT EXISTS users (
      id SERIAL PRIMARY KEY,
      name TEXT NOT NULL
  );
  ```

## Пример использования Go-клиента для 3x-ui

```go
import "TelegramXUI/internal/xui_client"

client := xui_client.NewClient(os.Getenv("XUI_URL"), os.Getenv("XUI_USER"), os.Getenv("XUI_PASSWORD"))
if err := client.Login(); err != nil {
    log.Fatal("Login error:", err)
}
users, err := client.GetUsers()
if err != nil {
    log.Fatal("GetUsers error:", err)
}
for _, user := range users {
    fmt.Println(user)
}
```

## Пример docker-compose.yml

```yaml
docker-compose.yml
version: '3.8'

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
      XUI_URL: "http://1.2.3.4:54321/fakepanel"
      XUI_USER: "fakeadmin"
      XUI_PASSWORD: "fakepass"
    command: ["air"]
    ports:
      - "25566:25566"
    volumes:
      - ./:/app
```

## Лицензия

MIT

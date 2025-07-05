# Отслеживание серверов XUI

## Описание

Добавлена система отслеживания серверов XUI с возможностью вести учет:
- Кто добавил сервер
- Когда был добавлен сервер
- Информация о серверах и их настройках подключения
- Статистика по серверам
- Система прав доступа с глобальным администратором

## Глобальный администратор

### Настройка

В Docker Compose необходимо указать глобального администратора:

```yaml
environment:
  # Telegram ID глобального администратора (обязательно)
  GLOBAL_ADMIN_TG_ID: "123456789"
  
  # Username глобального администратора (обязательно)
  GLOBAL_ADMIN_USERNAME: "your_admin_username"
```

### Как получить Telegram ID

1. Найдите @userinfobot в Telegram
2. Отправьте команду `/start`
3. Бот покажет ваш ID
4. Скопируйте ID в `GLOBAL_ADMIN_TG_ID`

### Права глобального администратора

Глобальный администратор имеет следующие права:
- ✅ Просмотр всех серверов в системе
- ✅ Управление всеми серверами (добавление, редактирование, удаление)
- ✅ Просмотр статистики по всем серверам
- ✅ Управление пользователями
- ✅ Доступ к административным функциям
- ✅ Просмотр серверов по датам
- ✅ Полная статистика системы

### Права обычных пользователей

Обычные пользователи могут:
- ✅ Добавлять свои серверы
- ✅ Просматривать свои серверы
- ❌ Просматривать серверы других пользователей
- ❌ Удалять чужие серверы
- ❌ Просматривать статистику системы

## Структура базы данных

### Таблица `xui_servers`

```sql
CREATE TABLE xui_servers (
    id SERIAL PRIMARY KEY,
    server_url VARCHAR(255) NOT NULL,
    server_name VARCHAR(255),
    server_location VARCHAR(255),
    server_ip VARCHAR(45),
    server_port INTEGER DEFAULT 54321,
    username VARCHAR(255),
    password VARCHAR(255),
    secret_key VARCHAR(255),
    two_factor_secret VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    added_by_tg_id BIGINT NOT NULL,
    added_by_username VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Поля:**
- `id` - уникальный идентификатор записи
- `server_url` - URL сервера XUI (например, https://server.com:54321)
- `server_name` - название сервера
- `server_location` - географическое расположение сервера
- `server_ip` - IP адрес сервера
- `server_port` - порт сервера (по умолчанию 54321)
- `username` - имя пользователя для подключения к XUI панели
- `password` - пароль для подключения к XUI панели
- `secret_key` - секретный ключ для аутентификации
- `two_factor_secret` - секретный ключ для двухфакторной аутентификации (2FA)
- `is_active` - активен ли сервер
- `added_by_tg_id` - Telegram ID пользователя, который добавил сервер
- `added_by_username` - username пользователя, который добавил сервер
- `created_at` - дата создания записи
- `updated_at` - дата последнего обновления

## API Endpoints

### 1. Получение серверов по пользователю
```
GET /api/servers/by-user?tg_id=123456789&username=user123
```

**Параметры:**
- `tg_id` - Telegram ID пользователя (обязательно)
- `username` - username пользователя (опционально)

**Ответ:**
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "server_url": "https://server1.com:54321",
      "server_name": "Server 1",
      "server_location": "Netherlands",
      "server_ip": "1.2.3.4",
      "server_port": 54321,
      "username": "admin",
      "password": "password123",
      "secret_key": "secret_key_here",
      "two_factor_secret": "2fa_secret_here",
      "is_active": true,
      "added_by_tg_id": 123456789,
      "added_by_username": "admin",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "count": 1,
  "permissions": {
    "is_global_admin": false,
    "can_manage_servers": false,
    "can_view_all_servers": false,
    "can_delete_servers": false,
    "can_manage_users": false,
    "can_view_stats": false,
    "can_add_servers": true,
    "can_view_own_servers": true
  }
}
```

### 2. Получение всех серверов (только для админов)
```
GET /api/servers/all?admin_tg_id=123456789&admin_username=admin&limit=20&offset=0
```

**Параметры:**
- `admin_tg_id` - Telegram ID администратора (обязательно)
- `admin_username` - username администратора (опционально)
- `limit` - количество записей на страницу (по умолчанию 50, максимум 100)
- `offset` - смещение от начала (по умолчанию 0)

### 3. Получение сервера по ID (с проверкой прав)
```
GET /api/servers/by-id?id=1&tg_id=123456789&username=user123
```

### 4. Получение сервера по URL (с проверкой прав)
```
GET /api/servers/by-url?url=https://server1.com:54321&tg_id=123456789&username=user123
```

### 5. Получение активных серверов (с проверкой прав)
```
GET /api/servers/active?tg_id=123456789&username=user123
```

### 6. Получение серверов по диапазону дат (только для админов)
```
GET /api/servers/by-date?admin_tg_id=123456789&admin_username=admin&start_date=2024-01-01&end_date=2024-01-31
```

**Формат дат:** YYYY-MM-DD

### 7. Статистика по серверам (только для админов)
```
GET /api/servers/stats?admin_tg_id=123456789&admin_username=admin
```

**Ответ:**
```json
{
  "success": true,
  "data": {
    "total_servers": 15,
    "active_servers": 12,
    "servers_last_30_days": 3,
    "servers_last_7_days": 1,
    "period": {
      "start_date": "2023-12-16",
      "end_date": "2024-01-15"
    },
    "admin_info": {
      "tg_id": 123456789,
      "username": "admin",
      "is_set": true
    }
  }
}
```

### 8. Информация о правах пользователя
```
GET /api/servers/admin-info?tg_id=123456789&username=user123
```

**Ответ:**
```json
{
  "success": true,
  "data": {
    "permissions": {
      "is_global_admin": false,
      "can_manage_servers": false,
      "can_view_all_servers": false,
      "can_delete_servers": false,
      "can_manage_users": false,
      "can_view_stats": false,
      "can_add_servers": true,
      "can_view_own_servers": true
    },
    "global_admin_info": {
      "tg_id": 123456789,
      "username": "admin",
      "is_set": true
    },
    "is_configured": true
  }
}
```

## Интеграция с существующим кодом

### Добавление записи о сервере

При добавлении нового сервера XUI необходимо добавить запись в таблицу:

```go
// Пример использования в сервисе
func (s *Service) AddServerWithTracking(serverURL, serverName, serverLocation, serverIP, 
    username, password, secretKey, twoFactorSecret string, serverPort int, 
    addedByTgID int64, addedByUsername string) error {
    
    // Добавляем запись в базу данных
    server := &services.XUIServer{
        ServerURL:       serverURL,
        ServerName:      serverName,
        ServerLocation:  serverLocation,
        ServerIP:        serverIP,
        ServerPort:      serverPort,
        Username:        username,
        Password:        password,
        SecretKey:       secretKey,
        TwoFactorSecret: twoFactorSecret,
        IsActive:        true,
        AddedByTgID:     addedByTgID,
        AddedByUsername: addedByUsername,
    }
    
    return s.serverService.AddServer(server)
}
```

### Использование сервиса администратора

```go
// Создание сервиса администратора
adminService := services.NewAdminService(config)

// Проверка прав
if adminService.IsGlobalAdmin(tgID) {
    // Пользователь является глобальным администратором
}

// Получение всех прав пользователя
permissions := adminService.GetUserPermissions(tgID, username)

if permissions["can_view_all_servers"] {
    // Пользователь может просматривать все серверы
}
```

## Безопасность

**⚠️ ВАЖНО:** Поля с паролями и секретными ключами содержат чувствительную информацию!

### Рекомендации по безопасности:

1. **Шифрование данных:**
   - Пароли и секретные ключи должны быть зашифрованы перед сохранением в базу
   - Используйте сильные алгоритмы шифрования (AES-256)

2. **Доступ к API:**
   - Ограничьте доступ к API endpoints, возвращающим пароли
   - Используйте аутентификацию и авторизацию
   - Логируйте все запросы к чувствительным данным

3. **Хранение в базе данных:**
   - Рассмотрите возможность использования внешних систем управления секретами
   - Регулярно ротируйте пароли и ключи
   - Ограничьте доступ к базе данных

4. **Глобальный администратор:**
   - Храните `GLOBAL_ADMIN_TG_ID` и `GLOBAL_ADMIN_USERNAME` в безопасном месте
   - Не передавайте эти данные третьим лицам
   - Регулярно проверяйте логи на подозрительную активность

5. **Пример шифрования паролей:**
```go
import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
)

// Шифрование пароля перед сохранением
func encryptPassword(password, key []byte) (string, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }
    
    ciphertext := make([]byte, aes.BlockSize+len(password))
    iv := ciphertext[:aes.BlockSize]
    if _, err := io.ReadFull(rand.Reader, iv); err != nil {
        return "", err
    }
    
    stream := cipher.NewCFBEncrypter(block, iv)
    stream.XORKeyStream(ciphertext[aes.BlockSize:], password)
    
    return base64.URLEncoding.EncodeToString(ciphertext), nil
}
```

## Миграции

Для применения изменений в базе данных выполните:

```bash
# Применение миграции
goose up

# Откат миграции (если нужно)
goose down
```

## Индексы

Созданы следующие индексы для оптимизации запросов:

- `idx_xui_servers_url` - поиск по URL сервера
- `idx_xui_servers_added_by_tg_id` - поиск по пользователю, который добавил
- `idx_xui_servers_ip` - поиск по IP сервера
- `idx_xui_servers_created_at` - поиск по дате создания

## Мониторинг

Система позволяет отслеживать:
- Кто и когда добавил серверы
- Активность пользователей
- Статистику по периодам
- Статус серверов (активные/неактивные)
- Информацию для подключения к серверам
- Права доступа пользователей

## Расширение функциональности

Возможные улучшения:
1. Экспорт данных в CSV/Excel (без паролей)
2. Уведомления о новых серверах
3. Автоматическая проверка доступности серверов
4. Интеграция с системой мониторинга
5. Графики и аналитика
6. Автоматическое определение IP адреса по URL
7. Система управления секретами (HashiCorp Vault, AWS Secrets Manager)
8. Автоматическая ротация паролей
9. Роли пользователей (админ, модератор, пользователь)
10. Временные права доступа 
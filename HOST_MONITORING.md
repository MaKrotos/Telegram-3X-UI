# 🔍 Мониторинг хостов XUI

Система автоматического мониторинга XUI хостов с уведомлениями администраторов.

## 📋 Обзор

Система мониторинга автоматически проверяет доступность всех добавленных XUI хостов и уведомляет администраторов о проблемах. Неактивные хосты автоматически отключаются и исключаются из пула для создания VPN подключений.

## ⚙️ Конфигурация

### Переменные окружения

```bash
# Telegram ID глобального администратора
GLOBAL_ADMIN_TG_ID=123456789

# Username глобального администратора
GLOBAL_ADMIN_USERNAME=admin_username

# Интервал проверки хостов (в минутах)
HOST_MONITOR_INTERVAL_MINUTES=5
```

### Настройка администратора

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

## 🔄 Процесс мониторинга

### Автоматические проверки

1. **Запуск**: Мониторинг запускается автоматически при старте приложения
2. **Интервал**: Проверки выполняются каждые N минут (настраивается)
3. **Параллельность**: Все хосты проверяются одновременно
4. **Таймаут**: Каждая проверка имеет ограничение по времени

### Проверки для каждого хоста

1. **Авторизация**:
   - Попытка входа в панель XUI
   - Проверка получения session token
   - Ошибка: "session token not found in cookies"

2. **API тестирование**:
   - Запрос списка пользователей
   - Проверка успешности ответа
   - Ошибка: "request not successful"

### Обработка результатов

- **Активный хост**: ✅ Продолжает работать
- **Неактивный хост**: 
  - 🔴 Помечается как неактивный в БД
  - 📧 Отправляется уведомление администратору
  - ❌ Исключается из пула VPN

## 📧 Уведомления

### Формат уведомления

```
🚨 ВНИМАНИЕ! Обнаружены неактивные хосты:

❌ Server Name (http://server:port)
   Ошибка: session token not found in cookies
   Проверено: 15.12.2024 14:30:25

❌ Another Server (http://another:port)
   Ошибка: request not successful
   Проверено: 15.12.2024 14:30:25

Хосты автоматически отключены и не будут использоваться для создания VPN.
```

### Получатели

- **Глобальные администраторы**: все пользователи с правами глобального админа
- **Канал уведомлений**: только в личные сообщения администраторам

## 🎛️ Управление мониторингом

### Команды администратора

#### `/monitor` - Общая информация
```
🔍 Мониторинг хостов

📊 Статус: 🟢 Запущен
⏱️ Интервал проверки: 5m0s

Команды управления:
/monitor_start - Запустить мониторинг
/monitor_stop - Остановить мониторинг
/monitor_status - Подробный статус
/check_hosts - Проверить все хосты сейчас
```

#### `/monitor_start` - Запуск мониторинга
```
🟢 Мониторинг хостов запущен!

Система будет автоматически проверять доступность всех активных хостов и уведомлять о проблемах.
```

#### `/monitor_stop` - Остановка мониторинга
```
🔴 Мониторинг хостов остановлен!

Автоматические проверки хостов прекращены.
```

#### `/monitor_status` - Детальная статистика
```
📊 Статус мониторинга хостов

🔄 Мониторинг: 🟢 Запущен
⏱️ Интервал проверки: 5m0s

📈 Статистика хостов:
🟢 Активных: 3
🔴 Неактивных: 1
📊 Всего: 4

⚠️ Неактивные хосты:
• Problem Server (http://problem:port)
• Another Problem (http://another:port)
```

#### `/check_hosts` - Немедленная проверка
```
🔍 Начинаем проверку всех активных хостов...

Это может занять некоторое время.
```

**Результат:**
```
📊 Результаты проверки хостов

✅ Проверено хостов: 4
🟢 Активных: 3
🔴 Неактивных: 1

Детали:
🟢 Server 1 - Активен
🟢 Server 2 - Активен
🟢 Server 3 - Активен
🔴 Problem Server - Неактивен (session token not found in cookies)

⚠️ Неактивные хосты автоматически отключены и не будут использоваться для создания VPN.
```

## 🗄️ База данных

### Таблица xui_servers

```sql
CREATE TABLE xui_servers (
    id SERIAL PRIMARY KEY,
    server_url TEXT NOT NULL,
    server_name TEXT NOT NULL,
    server_location TEXT,
    server_ip TEXT,
    server_port INTEGER,
    username TEXT NOT NULL,
    password TEXT NOT NULL,
    secret_key TEXT,
    two_factor_secret TEXT,
    is_active BOOLEAN DEFAULT true,
    added_by_tg_id BIGINT NOT NULL,
    added_by_username TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Поля мониторинга

- `is_active` - статус активности хоста
- `updated_at` - время последнего обновления (при изменении статуса)

## 🔧 Технические детали

### Архитектура

```
HostMonitorService
├── XUIServerService (проверка хостов)
├── AdminService (получение администраторов)
├── TelegramClient (отправка уведомлений)
└── Timer (периодические проверки)
```

### Логирование

```
[HostMonitor] Мониторинг хостов запущен с интервалом 5m0s
[HostMonitor] Начинаем проверку всех активных хостов
[HostMonitor] Проверяем хост: Server Name (http://server:port)
[HostMonitor] Хост Server Name активен
[HostMonitor] Хост Problem Server неактивен: session token not found in cookies
[HostMonitor] Проверка завершена. Найдено неактивных хостов: 1
[HostMonitor] Уведомление о неактивных хостах отправлено администратору 123456789
```

### Обработка ошибок

- **Ошибки подключения**: логируются, хост помечается как неактивный
- **Ошибки авторизации**: логируются, хост помечается как неактивный
- **Ошибки API**: логируются, хост помечается как неактивный
- **Ошибки уведомлений**: логируются, но не влияют на мониторинг

## 🚨 Troubleshooting

### Мониторинг не запускается

1. **Проверьте конфигурацию**:
   ```bash
   echo $GLOBAL_ADMIN_TG_ID
   echo $GLOBAL_ADMIN_USERNAME
   echo $HOST_MONITOR_INTERVAL_MINUTES
   ```

2. **Проверьте логи**:
   ```bash
   docker-compose logs app | grep HostMonitor
   ```

### Уведомления не приходят

1. **Проверьте права администратора**:
   - Убедитесь, что ваш Telegram ID указан правильно
   - Проверьте, что username совпадает

2. **Проверьте бота**:
   - Бот должен быть запущен
   - У бота должны быть права на отправку сообщений

### Хост помечается как неактивный

1. **Проверьте доступность**:
   ```bash
   curl -I http://your-server:port
   ```

2. **Проверьте учетные данные**:
   - Логин и пароль должны быть корректными
   - Пользователь должен иметь права администратора

3. **Проверьте 2FA**:
   - Если включена двухфакторная аутентификация, укажите secret_key

## 📈 Метрики

### Доступные метрики

- Количество активных хостов
- Количество неактивных хостов
- Время последней проверки
- Статус мониторинга (запущен/остановлен)
- Интервал проверки

### Получение метрик

```bash
# Через команду бота
/monitor_status

# Через API (планируется)
GET /v1/monitor/status
```

## 🔮 Планы развития

- [ ] Веб-интерфейс для управления мониторингом
- [ ] Графики и статистика доступности
- [ ] Настраиваемые пороги для уведомлений
- [ ] Интеграция с внешними системами мониторинга
- [ ] Автоматическое восстановление хостов
- [ ] Уведомления в Telegram каналы 
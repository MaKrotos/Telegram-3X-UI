# Расширяемая система состояний пользователей

## Обзор

Расширяемая система состояний позволяет добавлять новые состояния и действия пользователей на лету без изменения кода. Это достигается за счет хранения определений состояний и действий в базе данных вместо жестко заданных enum типов.

## Архитектура

### Основные компоненты

1. **Таблица `user_states`** - определения состояний
2. **Таблица `expected_actions`** - определения ожидаемых действий
3. **Таблица `state_action_mappings`** - связи состояний и действий
4. **Таблица `user_state_history`** - история изменений состояний
5. **Сервис `ExtensibleStateService`** - управление системой
6. **HTTP обработчики** - API для управления

### Преимущества расширяемой системы

- ✅ **Добавление новых состояний без перезапуска**
- ✅ **Гибкая настройка прав для каждого состояния**
- ✅ **Автоматическое истечение состояний**
- ✅ **История всех изменений**
- ✅ **Валидация комбинаций состояний и действий**
- ✅ **Веб-интерфейс для управления**

## Структура базы данных

### Таблица user_states

```sql
CREATE TABLE user_states (
    id SERIAL PRIMARY KEY,
    state_code VARCHAR(50) UNIQUE NOT NULL,        -- Код состояния (например, 'premium')
    state_name VARCHAR(100) NOT NULL,              -- Название состояния
    description TEXT,                              -- Описание
    is_active BOOLEAN DEFAULT TRUE,                -- Активно ли состояние
    can_perform_actions BOOLEAN DEFAULT FALSE,     -- Может выполнять действия
    can_manage_servers BOOLEAN DEFAULT FALSE,      -- Может управлять серверами
    can_create_connections BOOLEAN DEFAULT FALSE,  -- Может создавать соединения
    can_view_only BOOLEAN DEFAULT FALSE,           -- Только просмотр
    requires_admin_approval BOOLEAN DEFAULT FALSE, -- Требует одобрения админа
    auto_expire BOOLEAN DEFAULT FALSE,             -- Автоматически истекает
    default_expiry_duration INTERVAL,              -- Длительность по умолчанию
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Таблица expected_actions

```sql
CREATE TABLE expected_actions (
    id SERIAL PRIMARY KEY,
    action_code VARCHAR(50) UNIQUE NOT NULL,       -- Код действия
    action_name VARCHAR(100) NOT NULL,             -- Название действия
    description TEXT,                              -- Описание
    is_active BOOLEAN DEFAULT TRUE,                -- Активно ли действие
    priority INTEGER DEFAULT 0,                    -- Приоритет
    auto_resolve BOOLEAN DEFAULT FALSE,            -- Автоматически разрешается
    auto_resolve_after INTERVAL,                   -- Время автоматического разрешения
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Таблица state_action_mappings

```sql
CREATE TABLE state_action_mappings (
    id SERIAL PRIMARY KEY,
    state_code VARCHAR(50) NOT NULL REFERENCES user_states(state_code),
    action_code VARCHAR(50) NOT NULL REFERENCES expected_actions(action_code),
    is_default BOOLEAN DEFAULT FALSE,              -- Действие по умолчанию
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(state_code, action_code)
);
```

## API Endpoints

### Управление состояниями

#### Получение всех состояний
```http
GET /api/admin/states
```

#### Получение конкретного состояния
```http
GET /api/admin/states/{state_code}
```

#### Создание нового состояния
```http
POST /api/admin/states
Content-Type: application/json

{
  "state_code": "vip",
  "state_name": "VIP пользователь",
  "description": "Премиум доступ с расширенными возможностями",
  "can_perform_actions": true,
  "can_manage_servers": true,
  "can_create_connections": true,
  "can_view_only": false,
  "requires_admin_approval": false,
  "auto_expire": false
}
```

#### Обновление состояния
```http
PUT /api/admin/states/{state_code}
Content-Type: application/json

{
  "state_name": "VIP пользователь (обновлено)",
  "description": "Обновленное описание",
  "can_perform_actions": true,
  "can_manage_servers": true,
  "can_create_connections": true,
  "can_view_only": false,
  "requires_admin_approval": false,
  "auto_expire": false
}
```

#### Удаление состояния
```http
DELETE /api/admin/states/{state_code}
```

### Управление действиями

#### Получение всех действий
```http
GET /api/admin/actions
```

#### Получение конкретного действия
```http
GET /api/admin/actions/{action_code}
```

#### Создание нового действия
```http
POST /api/admin/actions
Content-Type: application/json

{
  "action_code": "verify_identity",
  "action_name": "Верификация личности",
  "description": "Пользователь должен пройти верификацию личности",
  "priority": 2,
  "auto_resolve": false
}
```

#### Обновление действия
```http
PUT /api/admin/actions/{action_code}
Content-Type: application/json

{
  "action_name": "Верификация личности (обновлено)",
  "description": "Обновленное описание",
  "priority": 1,
  "auto_resolve": false
}
```

#### Удаление действия
```http
DELETE /api/admin/actions/{action_code}
```

### Управление связями состояний и действий

#### Получение доступных действий для состояния
```http
GET /api/admin/states/{state_code}/actions
```

#### Установка действия по умолчанию
```http
PUT /api/admin/states/{state_code}/default-action/{action_code}
```

#### Проверка валидности комбинации
```http
GET /api/admin/states/{state_code}/validate-action/{action_code}
```

### История состояний

#### Получение истории пользователя
```http
GET /api/users/{telegram_id}/state-history?limit=50&offset=0
```

### Информация для управления

#### Полная информация о системе состояний
```http
GET /api/admin/state-management-info
```

**Ответ:**
```json
{
  "states": [
    {
      "id": 1,
      "state_code": "active",
      "state_name": "Активный",
      "description": "Полный доступ к системе",
      "is_active": true,
      "can_perform_actions": true,
      "can_manage_servers": true,
      "can_create_connections": true,
      "can_view_only": false,
      "requires_admin_approval": false,
      "auto_expire": false,
      "default_expiry_duration": null
    }
  ],
  "actions": [
    {
      "id": 1,
      "action_code": "none",
      "action_name": "Ничего не ожидается",
      "description": "Пользователь может использовать систему нормально",
      "is_active": true,
      "priority": 0,
      "auto_resolve": false,
      "auto_resolve_after": null
    }
  ],
  "state_action_mappings": {
    "active": ["none"],
    "pending_verification": ["verify_email", "complete_profile"],
    "blocked": ["contact_support"]
  }
}
```

## Примеры использования

### Добавление нового состояния "VIP"

```bash
# 1. Создаем новое состояние
curl -X POST http://localhost:8080/api/admin/states \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "state_code": "vip",
    "state_name": "VIP пользователь",
    "description": "Премиум доступ с расширенными возможностями",
    "can_perform_actions": true,
    "can_manage_servers": true,
    "can_create_connections": true,
    "can_view_only": false,
    "requires_admin_approval": false,
    "auto_expire": false
  }'

# 2. Создаем новое действие
curl -X POST http://localhost:8080/api/admin/actions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "action_code": "upgrade_to_vip",
    "action_name": "Обновление до VIP",
    "description": "Пользователь должен обновить свой план до VIP",
    "priority": 1,
    "auto_resolve": false
  }'

# 3. Связываем состояние и действие
curl -X PUT http://localhost:8080/api/admin/states/vip/default-action/upgrade_to_vip \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Добавление временного состояния "Пробный период"

```bash
# Создаем состояние с автоматическим истечением
curl -X POST http://localhost:8080/api/admin/states \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "state_code": "trial",
    "state_name": "Пробный период",
    "description": "Ограниченный доступ на время пробного периода",
    "can_perform_actions": true,
    "can_manage_servers": true,
    "can_create_connections": true,
    "can_view_only": false,
    "requires_admin_approval": false,
    "auto_expire": true,
    "default_expiry_duration": "30 days"
  }'
```

### Добавление состояния с автоматическим разрешением действия

```bash
# Создаем действие с автоматическим разрешением
curl -X POST http://localhost:8080/api/admin/actions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "action_code": "wait_activation",
    "action_name": "Ожидание активации",
    "description": "Пользователь ждет автоматической активации",
    "priority": 0,
    "auto_resolve": true,
    "auto_resolve_after": "24 hours"
  }'
```

## Использование в коде

### Инициализация сервиса

```go
import "Telegram-3X-UI/internal/services"

// Создание сервиса расширяемых состояний
extensibleStateService := services.NewExtensibleStateService(db)

// Создание обработчика
extensibleStateHandler := handlers.NewExtensibleStateHandler(extensibleStateService, adminService)
```

### Проверка прав пользователя

```go
// Получаем определение состояния пользователя
stateDef, err := extensibleStateService.GetStateDefinition(userState.State)
if err != nil {
    return err
}

if stateDef == nil {
    return fmt.Errorf("неизвестное состояние: %s", userState.State)
}

// Проверяем права на управление серверами
if !stateDef.CanManageServers {
    return fmt.Errorf("пользователь не может управлять серверами в состоянии %s", userState.State)
}

// Проверяем права на создание соединений
if !stateDef.CanCreateConnections {
    return fmt.Errorf("пользователь не может создавать соединения в состоянии %s", userState.State)
}
```

### Создание нового состояния программно

```go
// Создаем новое состояние
newState := &services.StateDefinition{
    StateCode:             "premium_plus",
    StateName:             "Premium Plus",
    Description:           "Расширенный премиум доступ",
    IsActive:              true,
    CanPerformActions:     true,
    CanManageServers:      true,
    CanCreateConnections:  true,
    CanViewOnly:           false,
    RequiresAdminApproval: false,
    AutoExpire:            false,
}

err := extensibleStateService.CreateStateDefinition(newState)
if err != nil {
    log.Printf("Ошибка создания состояния: %v", err)
}
```

### Получение доступных действий для состояния

```go
// Получаем доступные действия для состояния
actions, err := extensibleStateService.GetAvailableActionsForState("pending_verification")
if err != nil {
    return err
}

// Показываем пользователю доступные действия
for _, action := range actions {
    fmt.Printf("Доступное действие: %s - %s\n", action.ActionName, action.Description)
}
```

## Автоматические процессы

### Обработка истекших состояний

```go
// Функция для обработки истекших состояний
func processExpiredStates(extensibleStateService *services.ExtensibleStateService) {
    // Получаем пользователей с истекшими состояниями
    expiredUsers, err := userStateService.GetExpiredStates()
    if err != nil {
        log.Printf("Ошибка получения истекших состояний: %v", err)
        return
    }
    
    for _, user := range expiredUsers {
        // Получаем определение состояния
        stateDef, err := extensibleStateService.GetStateDefinition(user.State)
        if err != nil {
            continue
        }
        
        if stateDef != nil && stateDef.AutoExpire {
            // Автоматически переводим в активное состояние
            err := userStateService.ActivateUser(
                user.TelegramID,
                0, // system
                "system",
            )
            if err != nil {
                log.Printf("Ошибка активации пользователя %d: %v", user.TelegramID, err)
            }
        }
    }
}
```

### Автоматическое разрешение действий

```go
// Функция для автоматического разрешения действий
func processAutoResolveActions(extensibleStateService *services.ExtensibleStateService) {
    // Получаем все действия с автоматическим разрешением
    actions, err := extensibleStateService.GetAllActionDefinitions()
    if err != nil {
        return
    }
    
    for _, action := range actions {
        if action.AutoResolve && action.AutoResolveAfter != nil {
            // Логика автоматического разрешения
            // ...
        }
    }
}
```

## Мониторинг и уведомления

### Метрики для отслеживания

1. **Количество пользователей по состояниям**
2. **Время нахождения в каждом состоянии**
3. **Частота изменений состояний**
4. **Количество истекших состояний**
5. **Эффективность автоматических процессов**

### Рекомендуемые уведомления

1. **Новые состояния** - уведомление администраторов о создании
2. **Изменения состояний** - логирование всех изменений
3. **Истекшие состояния** - уведомления о необходимости действий
4. **Ошибки валидации** - уведомления о невалидных комбинациях

## Безопасность

### Рекомендации

1. **Валидация входных данных** - проверка всех создаваемых состояний
2. **Ограничение доступа** - только администраторы могут управлять состояниями
3. **Аудит изменений** - ведение полной истории изменений
4. **Резервное копирование** - регулярное резервное копирование определений
5. **Тестирование** - тестирование новых состояний в тестовой среде

### Проверки безопасности

```go
// Проверка валидности комбинации состояния и действия
valid, err := extensibleStateService.ValidateStateActionCombination(stateCode, actionCode)
if err != nil {
    return err
}

if !valid {
    return fmt.Errorf("невалидная комбинация состояния %s и действия %s", stateCode, actionCode)
}
```

## Миграция с enum на расширяемую систему

### Пошаговый процесс

1. **Применение миграций**
   ```bash
   # Применяем миграции по порядку
   goose up
   ```

2. **Проверка данных**
   ```sql
   -- Проверяем, что все состояния перенесены
   SELECT * FROM user_states;
   
   -- Проверяем связи
   SELECT * FROM state_action_mappings;
   ```

3. **Обновление кода**
   ```go
   // Заменяем старый сервис на новый
   // userStateService -> extensibleStateService
   ```

4. **Тестирование**
   ```bash
   # Тестируем создание новых состояний
   curl -X POST http://localhost:8080/api/admin/states \
     -H "Content-Type: application/json" \
     -d '{"state_code": "test", "state_name": "Тестовое состояние"}'
   ```

## Примеры новых состояний

### Состояния для разных типов пользователей

```json
{
  "state_code": "enterprise",
  "state_name": "Корпоративный",
  "description": "Корпоративный доступ с расширенными возможностями",
  "can_perform_actions": true,
  "can_manage_servers": true,
  "can_create_connections": true,
  "can_view_only": false,
  "requires_admin_approval": false,
  "auto_expire": false
}
```

```json
{
  "state_code": "student",
  "state_name": "Студент",
  "description": "Студенческий доступ с ограничениями",
  "can_perform_actions": true,
  "can_manage_servers": false,
  "can_create_connections": true,
  "can_view_only": false,
  "requires_admin_approval": false,
  "auto_expire": true,
  "default_expiry_duration": "4 months"
}
```

```json
{
  "state_code": "moderator",
  "state_name": "Модератор",
  "description": "Модератор с правами управления пользователями",
  "can_perform_actions": true,
  "can_manage_servers": true,
  "can_create_connections": true,
  "can_view_only": false,
  "requires_admin_approval": true,
  "auto_expire": false
}
```

### Состояния для специальных случаев

```json
{
  "state_code": "maintenance",
  "state_name": "Техническое обслуживание",
  "description": "Временное ограничение доступа",
  "can_perform_actions": false,
  "can_manage_servers": false,
  "can_create_connections": false,
  "can_view_only": true,
  "requires_admin_approval": false,
  "auto_expire": true,
  "default_expiry_duration": "2 hours"
}
```

```json
{
  "state_code": "fraud_check",
  "state_name": "Проверка на мошенничество",
  "description": "Временное ограничение для проверки",
  "can_perform_actions": false,
  "can_manage_servers": false,
  "can_create_connections": false,
  "can_view_only": true,
  "requires_admin_approval": true,
  "auto_expire": true,
  "default_expiry_duration": "48 hours"
}
```

Эта расширяемая система позволяет легко адаптироваться к новым требованиям бизнеса без необходимости изменения кода приложения. 
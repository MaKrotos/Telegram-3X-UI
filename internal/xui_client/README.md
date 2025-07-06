# Go-клиент для 3x-ui API

Минимальный пример работы с REST API панели 3x-ui на Go.

## Быстрый старт

1. Импортируйте пакет:

   ```go
   import "your_project/internal/xui_client"
   ```

2. Пример использования:

   ```go
   package main

   import (
       "fmt"
       "log"
       "your_project/internal/xui_client"
   )

   func main() {
       client := xui_client.NewClient("http://127.0.0.1:54321", "admin", "admin")
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
   }
   ```

3. Для других методов (добавление пользователя, inbound и т.д.) — пишите, добавлю примеры!

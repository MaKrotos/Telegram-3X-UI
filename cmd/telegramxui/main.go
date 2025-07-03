package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"

	"TelegramXUI/internal/xui_client"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func getUsersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Попытка подключения к x-ui
		client := xui_client.NewClient(
			os.Getenv("XUI_URL"),
			os.Getenv("XUI_USER"),
			os.Getenv("XUI_PASSWORD"),
		)
		var xuiStatus string
		if err := client.Login(); err != nil {
			xuiStatus = "Ошибка подключения к x-ui: " + err.Error()
		} else {
			xuiStatus = "Успешное подключение к x-ui"
		}

		rows, err := db.Query("SELECT id, name FROM users")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Ошибка получения пользователей: " + err.Error()))
			return
		}
		defer rows.Close()
		var users []User
		for rows.Next() {
			var u User
			if err := rows.Scan(&u.ID, &u.Name); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Ошибка сканирования пользователя: " + err.Error()))
				return
			}
			users = append(users, u)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"xui_status": xuiStatus,
			"users":      users,
		})
	}
}

func newXUIClient() *xui_client.Client {
	return xui_client.NewClient("http://37.46.19.85:25567/vLr9dnLbg0B140e", "MaKrotos", "3483hiT7")
}

func main() {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://user:password@localhost:5432/telegramxui?sslmode=disable"
		fmt.Println("POSTGRES_DSN не задан, используется стандартный: ", dsn)
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к Postgres: %v", err)
	}
	defer db.Close()

	if err := goose.Up(db, "internal/migrations"); err != nil {
		log.Fatalf("Ошибка применения миграций: %v", err)
	}

	client := newXUIClient()
	if err := client.Login(); err != nil {
		log.Fatal("Login error:", err)
	}

	// Используем случайный порт для inbound
	port := 20000 + rand.Intn(40000)
	emptyInbound := xui_client.GenerateEmptyInboundForm(port, "inbound без пользователей")
	inboundId, err := client.AddInbound(emptyInbound)
	if err != nil || inboundId == 0 {
		log.Fatalf("Ошибка добавления пустого inbound или порт занят! id=%d err=%v", inboundId, err)
	}
	fmt.Println("[main] Inbound без пользователей успешно добавлен! id=", inboundId)

	clientId, email, subId, settings := xui_client.GenerateRandomClientSettings(10)
	addClientForm := &xui_client.AddClientForm{
		Id:       inboundId,
		Settings: settings,
	}
	if err := client.AddClientToInbound(addClientForm); err != nil {
		log.Fatal("Ошибка добавления клиента:", err)
	}
	fmt.Println("[main] Случайный клиент успешно добавлен в inbound! id=", clientId, " email=", email, " subId=", subId)

	http.HandleFunc("/v1/getUsers", getUsersHandler(db))
	log.Println("Сервер запущен на :25566")
	log.Fatal(http.ListenAndServe(":25566", nil))
}

package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

type PostgresBase struct {
	DB *sql.DB
}

func (b *PostgresBase) Connect() error {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		return fmt.Errorf("POSTGRES_DSN не задан")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("ошибка подключения к Postgres: %w", err)
	}
	b.DB = db
	return nil
}

func (b *PostgresBase) Close() error {
	if b.DB != nil {
		return b.DB.Close()
	}
	return nil
}

package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var DB *sql.DB

func Init(databaseDSN string) error {
	connection := getConnect(databaseDSN)

	var err error
	DB, err = sql.Open("pgx", connection)
	if err != nil {
		return fmt.Errorf("не удалось подключиться к bd postgres: %w", err)
	}
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("поверка подключения к bd postgres не удачно: %w", err)
	}
	return nil
}

func getConnect(connectionFlag string) string {
	if envConnection := os.Getenv("DATABASE_DSN"); envConnection != "" {
		return strings.Trim(envConnection, `"`)
	}
	if connectionFlag != "" {
		return strings.Trim(connectionFlag, `"`)
	}
	return "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
}

func Ping() error {
	if DB == nil {
		return fmt.Errorf("подключение к базе данных postgres отсутствует")
	}
	return DB.Ping()
}

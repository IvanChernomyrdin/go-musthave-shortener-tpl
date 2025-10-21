package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	migratePostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

func GetDB() *sql.DB {
	return DB
}

func RunMigrations(db *sql.DB) error {
	//создаём драйвер миграции
	driver, err := migratePostgres.WithInstance(db, &migratePostgres.Config{})
	if err != nil {
		return fmt.Errorf("ошибка создания драйвера миграции: %w", err)
	}
	//создаём мигратор
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("ошибка применения миграций: %w", err)
	}

	//применяем миграции
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("ошибка применения миграций: %w", err)
	}
	log.Println("Миграции были применены!")
	return nil
}

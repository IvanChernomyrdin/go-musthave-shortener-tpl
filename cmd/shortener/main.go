package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	config "github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/config"
	"github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/config/db"
	handler "github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/handler"
	middleware "github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/middleware"
	"github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/repository/postgres"
	storage "github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/storage"
	chi "github.com/go-chi/chi/v5"
)

func main() {
	//подключаем конфиг и проверяем его целостность данных
	cfg := config.NewConfig()
	cfg.Validate()

	var store storage.URLStorage

	// Если есть бд берём бд, или берём файл
	if cfg.DatabaseDSN != "" {
		if err := db.Init(cfg.DatabaseDSN); err != nil {
			log.Printf("Database unavailable, using file storage: %v", err)
		} else {
			defer db.DB.Close()

			// Запускаем миграции
			if err := db.RunMigrations(db.DB); err != nil {
				log.Printf("Migrations failed: %v", err)
			}

			// Используем PostgreSQL storage (НЕ игнорируем ошибку!)
			postgresStore, err := postgres.NewPostgresStorage(db.DB, cfg.BaseURL)
			if err != nil {
				log.Printf("Failed to create postgres storage: %v", err)
			} else {
				store = postgresStore
				log.Println("Using PostgreSQL storage")
			}
		}
	}

	if store == nil {
		var err error
		store, err = storage.NewFileStorage(cfg.FileStorage, cfg.BaseURL)
		if err != nil {
			log.Printf("Failed to create file storage: %v", err)
		}
		log.Println("Using file storage")
	}

	handler := handler.NewHandler(store, cfg.BaseURL)

	r := chi.NewRouter()

	loggerMiddleware, _ := middleware.NewLogger()
	defer loggerMiddleware.Logger.Sync()

	// gzip компрессия и декомпрессия
	r.Use(middleware.GzipDecompressMiddleware)
	r.Use(middleware.GzipCompressMiddleware)
	// логирование
	r.Use(loggerMiddleware.LoggingMiddleware)
	// переход по оригинальной ссылке
	r.Get("/{id}", handler.RedirectURL)
	// проверка подключения db postgres
	r.Get("/ping", handler.PingPostgres)
	// создание коротного url
	r.Post("/", handler.CreateShortURL)
	// {"url":"<some_url>"} получает и отдаёт {"result":"<short_url>"}
	r.Post("/api/shorten", handler.CreateShortURLJson)
	// принимает множество url для сокращения
	r.Post("/api/shorten/batch", handler.CreateShortURLBatch)

	// все остальные запросы
	r.NotFound(handler.NotFoundHandler)
	r.MethodNotAllowed(handler.MethodNotAllowedHandler)

	//создаём сервер и передаём туда наш chi
	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: r,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	//ждём уведление в канал
	<-quit
	log.Println("Gracefully shutdown server...")

	//создаём контекст с 5секундный таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//тушим сервер
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server force to shudown: %v", err)
	}
}

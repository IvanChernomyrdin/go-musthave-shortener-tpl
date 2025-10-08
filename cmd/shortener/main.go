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
	handler "github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/handler"
	middleware "github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/middleware"
	storage "github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/storage"
	chi "github.com/go-chi/chi/v5"
)

func main() {
	//подключаем конфиг и проверяем его целостность данных
	cfg := config.NewConfig()
	cfg.Validate()

	storage := storage.NewMemoryStorage()
	handler := handler.NewHandler(storage, cfg.BaseURL)

	r := chi.NewRouter()

	loggerMiddleware, _ := middleware.NewLogger()
	defer loggerMiddleware.Logger.Sync()

	r.Use(loggerMiddleware.LoggingMiddleware)
	// переход по оригинальной ссылке
	r.Get("/{id}", handler.RedirectURL)
	// создание коротного url
	r.Post("/", handler.CreateShortURL)

	// все остальные запросы
	r.NotFound(handler.NotFoundHandler)
	r.MethodNotAllowed(handler.MethodNotAllowedHandler)

	//создаём червер и передаёт туда наш chi
	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: r,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Server error: %v", err)
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
		log.Fatalf("Server force to shudown: %v", err)
	}
}

package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type Logger struct {
	Logger *zap.Logger
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int64
}

func NewLogger() (*Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return &Logger{Logger: logger}, nil
}

func (l *Logger) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//засекаем время
		start := time.Now()
		//создаём кастомный responseWriter
		wrapperWriter := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			size:           0,
		}
		//обрабатываем запрос
		next.ServeHTTP(wrapperWriter, r)
		//фиксируем время запроса
		duration := time.Since(start)

		l.Logger.Info("HTTP request",
			zap.String("method", r.Method),
			zap.String("url", r.RequestURI),
			zap.Duration("duration", duration),
			zap.Int("status", wrapperWriter.statusCode),
			zap.Int64("size", wrapperWriter.size),
		)
	})
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += int64(size)
	return size, err
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

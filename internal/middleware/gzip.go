package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// по тз
// при получении должен быть Content-Encoding
// при отправки должен быть Accept-Encoding

func GzipDecompressMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Invalid gzip format", http.StatusBadRequest)
			}
			defer gz.Close()
			// перезаписали распакованные данные в body
			r.Body = gz
		}
		next.ServeHTTP(w, r)
	})
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func GzipCompressMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			// ставим уровень и сжимаем
			gz, err := gzip.NewWriterLevel(w, gzip.DefaultCompression)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer gz.Close()

			w.Header().Set("Content-Encoding", "gzip")
			w = gzipResponseWriter{Writer: gz, ResponseWriter: w}
		}
		next.ServeHTTP(w, r)
	})
}

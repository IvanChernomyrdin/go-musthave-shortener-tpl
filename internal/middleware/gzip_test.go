package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGzipCompression(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":"http://localhost:8080/qwe1"}`))
	})
	tests := []struct {
		name           string
		gzip           bool
		acceptEncoding string
	}{
		{
			name:           "client access gzip",
			gzip:           true,
			acceptEncoding: "gzip",
		},
		{
			name:           "client don't access gzip",
			gzip:           false,
			acceptEncoding: "deflate",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/shorten", nil)
			req.Header.Set("Accept-Encoding", test.acceptEncoding)

			res := httptest.NewRecorder()

			middleware := GzipCompressMiddleware(handler)
			middleware.ServeHTTP(res, req)

			contentEncoding := res.Header().Get("Content-Encoding")
			if test.gzip && contentEncoding != "gzip" {
				t.Errorf("Expected Content-Encoding: gzip, got: %s", contentEncoding)
			}
			if !test.gzip && contentEncoding != "" {
				t.Errorf("Expected Content-Encoding, got: %s", contentEncoding)
			}
			if test.gzip {
				// распаковываем и делаем декомпрессию
				reader, err := gzip.NewReader(res.Body)
				if err != nil {
					t.Errorf("Response should be valid gzip: %v", err)
				}
				defer reader.Close()

				//проверяем что декомпрессия прошла удачно
				uncopression, err := io.ReadAll(reader)
				if err != nil {
					t.Errorf("Failed to decompression: %v", err)
				}
				expectUncopression := `{"result":"http://localhost:8080/qwe1"}`

				if string(uncopression) != expectUncopression {
					t.Fatalf("Expected %s, got %s", expectUncopression, string(uncopression))
				}
			}
		})
	}
}

func TestGzipDecompression(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Write(body)
	})

	tests := []struct {
		name            string
		contentEncoding string
		body            string
		compressBody    bool
	}{
		{
			name:            "gzip compress request",
			contentEncoding: "gzip",
			body:            `{"result":"http://localhost:8080/jopa777"}`,
			compressBody:    true,
		},
		{
			name:            "not gzip compress",
			contentEncoding: "",
			body:            `{"result":"http://localhost:8080/jopa666"}`,
			compressBody:    false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var bodyBytes []byte

			if test.compressBody {
				var buf bytes.Buffer
				gz := gzip.NewWriter(&buf)
				gz.Write([]byte(test.body))
				gz.Close()
				bodyBytes = buf.Bytes()
			} else {
				bodyBytes = []byte(test.body)
			}
			req := httptest.NewRequest("POST", "/api/shorten", bytes.NewReader(bodyBytes))

			res := httptest.NewRecorder()
			req.Header.Set("Content-Encoding", test.contentEncoding)

			middleware := GzipDecompressMiddleware(handler)
			middleware.ServeHTTP(res, req)

			if res.Body.String() != test.body {
				t.Errorf("Expected body: %s, got: %s", test.body, res.Body.String())
			}

		})
	}
}

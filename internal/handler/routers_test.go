package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	config "github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/config"
	"github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/model"
	storage "github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockStorage struct {
	urls map[string]string
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		urls: map[string]string{
			"1": "https://ya.ru",
			"2": "https://google.com",
		},
	}
}

func (m *MockStorage) Get(id string) (string, bool) {
	url, exists := m.urls[id]
	return url, exists
}
func (m *MockStorage) Save(url string) string {
	return ""
}

func TestRouters(t *testing.T) {
	mockStorage := storage.NewMemoryStorage()
	handler := NewHandler(mockStorage, config.HOST)

	//создаю роутер как в main
	r := chi.NewRouter()
	r.Post("/", handler.CreateShortURL)
	r.Get("/{id}", handler.RedirectURL)
	r.NotFound(handler.NotFoundHandler)
	r.MethodNotAllowed(handler.NotFoundHandler)

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{
			name:           "POST / with valid URL",
			method:         "POST",
			path:           "/",
			body:           "https://example.com",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "POST / with empty body",
			method:         "POST",
			path:           "/",
			body:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "GET /{id} with existing ID",
			method:         "GET",
			path:           "/1",
			expectedStatus: http.StatusTemporaryRedirect,
		},
		{
			name:           "GET /{id} with non-existing ID",
			method:         "GET",
			path:           "/999",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "GET / with empty ID",
			method:         "GET",
			path:           "/",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "PUT method not allowed",
			method:         "PUT",
			path:           "/",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Unknown path",
			method:         "GET",
			path:           "/unknown/path",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tt.method == "POST" {
				req, err = http.NewRequest(tt.method, config.HOST+tt.path, strings.NewReader(tt.body))
				require.NoError(t, err)
				req.Header.Set("Content-Type", "text/plain")
			} else {
				req, err = http.NewRequest(tt.method, config.HOST+tt.path, nil)
				require.NoError(t, err)
			}

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code,
				"Expected status %d, got %d for %s %s",
				tt.expectedStatus, rr.Code, tt.method, tt.path)
		})
	}
}

func TestCreateTestURL(t *testing.T) {
	//создали хранилище
	mockStorage := storage.NewMemoryStorage()
	//создали handler
	handler := NewHandler(mockStorage, config.HOST)

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
		expectedBody   string
		checkHeader    bool
	}{
		{
			name:           "Successful URL create",
			method:         http.MethodPost,
			body:           "https://example.com",
			expectedStatus: http.StatusCreated,
			expectedBody:   "http://localhost:8080/1",
			checkHeader:    true,
		},
		{
			name:           "Empty URL",
			method:         http.MethodPost,
			body:           "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "URL not be empty",
			checkHeader:    false,
		},
		{
			name:           "Wrong HTTP method",
			method:         http.MethodGet,
			body:           "http://example.com",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
			checkHeader:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest(test.method, "/", bytes.NewBufferString(test.body))
			if err != nil {
				t.Fatalf("Couldn't create request: %v", err)
			}
			rr := httptest.NewRecorder()

			handler.CreateShortURL(rr, req)

			if rr.Code != test.expectedStatus {
				t.Errorf("Handler return wrong status: %v", err)
			}
			if strings.TrimSpace(rr.Body.String()) != test.expectedBody {
				t.Errorf("Handler return unexpected body: got %v, want %v", rr.Body.String(), test.expectedBody)
			}

			if test.checkHeader {
				contentType := rr.Header().Get("Content-Type")
				expectedContentType := "text/plain"
				if contentType != expectedContentType {
					t.Errorf("Handler return wrong content type: got %v, want %v", contentType, expectedContentType)
				}
			}
		})
	}
}

func TestRedirectURL(t *testing.T) {
	mockStorage := NewMockStorage()
	handler := &Handler{
		storage: mockStorage,
		baseURL: config.HOST,
	}

	tests := []struct {
		name             string
		method           string
		path             string
		expectedStatus   int
		expectedLocation string
		expectedBody     string
	}{
		{
			name:             "first Successful redirect",
			method:           http.MethodGet,
			path:             "/1",
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedLocation: "https://ya.ru",
			expectedBody:     "",
		},
		{
			name:             "second Successful redirect",
			method:           http.MethodGet,
			path:             "/2",
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedLocation: "https://google.com",
			expectedBody:     "",
		},
		{
			name:             "Method not allowed",
			method:           http.MethodPost,
			path:             "/1",
			expectedStatus:   http.StatusMethodNotAllowed,
			expectedLocation: "",
			expectedBody:     "Method not allowed",
		},
		{
			name:             "Empty ID",
			method:           http.MethodGet,
			path:             "/",
			expectedStatus:   http.StatusBadRequest,
			expectedLocation: "",
			expectedBody:     "ID cannot be empty",
		},
		{
			name:             "URL not found",
			method:           http.MethodGet,
			path:             "/4",
			expectedStatus:   http.StatusNotFound,
			expectedLocation: "",
			expectedBody:     "URL not found",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest(test.method, test.path, nil)
			if err != nil {
				t.Fatalf("Couldn't create request: %v", err)
			}
			rr := httptest.NewRecorder()
			handler.RedirectURL(rr, req)

			if rr.Code != test.expectedStatus {
				t.Errorf("Handler return unexpected status: got %v, want %v", rr.Code, test.expectedStatus)
			}
			if test.expectedLocation != "" {
				location := rr.Header().Get("Location")
				if location != test.expectedLocation {
					t.Errorf("Handler return wrong location in header: got %v, want %v", location, test.expectedLocation)
				}
			}
			body := strings.TrimSpace(rr.Body.String())
			if test.expectedBody != "" && body != test.expectedBody {
				t.Errorf("Handler return unexpected body: got '%v', want '%v'", body, test.expectedBody)
			}
		})
	}
}

func TestCreateShortURLJson(t *testing.T) {
	//создали хранилище
	mockStorage := storage.NewMemoryStorage()
	//создали handler
	handler := NewHandler(mockStorage, config.HOST)

	tests := []struct {
		name       string
		request    model.ShortUrlJson // запрос
		wantStatus int                // ожидаемый статус
		wantError  bool               // ожидаем ошибку?
	}{
		{
			name:       "успешный запрос",
			request:    model.ShortUrlJson{URL: "https://yandex.ru"},
			wantStatus: http.StatusCreated,
			wantError:  false,
		},
		{
			name:       "пустой URL",
			request:    model.ShortUrlJson{URL: ""},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "URL с пробелами",
			request:    model.ShortUrlJson{URL: "   "},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, _ := json.Marshal(test.request)

			// создаём запрос
			req := httptest.NewRequest("POST", "/api/shorten", bytes.NewReader(body))
			// заголовок json
			req.Header.Set("Content-Type", "application/json")
			// записываем всё это в recover
			res := httptest.NewRecorder()
			// выполняем функцию с нашими данными
			handler.CreateShortURLJson(res, req)

			// проверяем по ошибкам
			if res.Code != test.wantStatus {
				t.Errorf("Status got: %v, want: %v", res.Code, test.wantStatus)
			}
			if !test.wantError {
				contentType := res.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Error("Content-Type should be application/json")
				}
			}
			if !test.wantError && res.Code == test.wantStatus {
				var response model.ShortUrlJsonResult
				if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
					t.Error("JSON not valid")
				}
				if response.Result == "" {
					t.Error("Response result not be empty")
				}
			}
		})
	}
}

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	config "github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/config"
	"github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/middleware"
	"github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/model"
	storage "github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockStorage struct {
	urls map[string]URLRecordMock
}

type URLRecordMock struct {
	OriginalURL string
	UserID      string
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		urls: map[string]URLRecordMock{
			"1": {
				OriginalURL: "https://ya.ru",
				UserID:      "user",
			},
			"2": {
				OriginalURL: "https://google.com",
				UserID:      "test",
			},
		},
	}
}

func (m *MockStorage) Get(id string) (string, bool) {
	url, exists := m.urls[id]
	return url.OriginalURL, exists
}

func (m *MockStorage) Save(url, userID string) (string, bool) {
	// Простая реализация для тестов
	for id, existingURL := range m.urls {
		if existingURL.OriginalURL == url {
			return id, true // конфликт
		}
	}

	// Генерируем новый ID
	newID := strconv.Itoa(len(m.urls) + 1)
	m.urls[newID] = URLRecordMock{
		OriginalURL: url,
		UserID:      userID,
	}
	return newID, false // нет конфликта
}

func (m *MockStorage) GetURLByUser(userID string) ([]storage.OriginalAndShortURLs, error) {
	var result []storage.OriginalAndShortURLs
	for id, record := range m.urls {
		if record.UserID == userID {
			result = append(result, storage.OriginalAndShortURLs{
				ShortURL:    "http://localhost:8080/" + id,
				OriginalURL: record.OriginalURL,
			})
		}
	}
	return result, nil
}

func (m *MockStorage) Ping() error {
	return nil
}

func TestRouters(t *testing.T) {
	mockStorage := NewMockStorage()
	handler := NewHandler(mockStorage, config.HOST)

	// создаю роутер как в main
	r := chi.NewRouter()
	r.Post("/", handler.CreateShortURL)
	r.Get("/{id}", handler.RedirectURL)
	r.Post("/api/shorten", handler.CreateShortURLJson)
	r.Get("/api/user/urls", handler.GetURLByUser) // Добавил новый эндпоинт
	r.NotFound(handler.NotFoundHandler)
	r.MethodNotAllowed(handler.MethodNotAllowedHandler)

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		userID         string // Добавил userID для контекста
		expectedStatus int
	}{
		{
			name:           "POST / with valid URL",
			method:         "POST",
			path:           "/",
			body:           "https://example.com",
			userID:         "test-user-1",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "POST / with empty body",
			method:         "POST",
			path:           "/",
			body:           "",
			userID:         "test-user-1",
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
		{
			name:           "GET /api/user/urls with user URLs",
			method:         "GET",
			path:           "/api/user/urls",
			userID:         "user", // У этого пользователя есть URL в моке
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET /api/user/urls with no user URLs",
			method:         "GET",
			path:           "/api/user/urls",
			userID:         "unknown-user", // У этого пользователя нет URL
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "GET /api/user/urls without user ID",
			method:         "GET",
			path:           "/api/user/urls",
			expectedStatus: http.StatusUnauthorized,
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

			// Добавляем userID в контекст если он указан
			if tt.userID != "" {
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
				req = req.WithContext(ctx)
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
	// создали хранилище
	mockStorage := NewMockStorage()
	// создали handler
	handler := NewHandler(mockStorage, config.HOST)

	tests := []struct {
		name           string
		method         string
		body           string
		userID         string
		expectedStatus int
		expectedBody   string
		checkHeader    bool
	}{
		{
			name:           "Successful URL create",
			method:         http.MethodPost,
			body:           "https://example.com",
			userID:         "test-user-1",
			expectedStatus: http.StatusCreated,
			expectedBody:   "http://localhost:8080/3", // Следующий ID после существующих
			checkHeader:    true,
		},
		{
			name:           "Empty URL",
			method:         http.MethodPost,
			body:           "",
			userID:         "test-user-1",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "URL not be empty",
			checkHeader:    false,
		},
		{
			name:           "Wrong HTTP method",
			method:         http.MethodGet,
			body:           "http://example.com",
			userID:         "test-user-1",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
			checkHeader:    false,
		},
		{
			name:           "Create URL without user ID",
			method:         http.MethodPost,
			body:           "https://example.com",
			userID:         "", // Нет userID
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "User not authenticated",
			checkHeader:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest(test.method, "/", bytes.NewBufferString(test.body))
			require.NoError(t, err)

			// Добавляем userID в контекст если он указан
			if test.userID != "" {
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, test.userID)
				req = req.WithContext(ctx)
			}

			rr := httptest.NewRecorder()
			handler.CreateShortURL(rr, req)

			assert.Equal(t, test.expectedStatus, rr.Code,
				"Handler returned wrong status: got %d, want %d", rr.Code, test.expectedStatus)

			if strings.TrimSpace(rr.Body.String()) != test.expectedBody {
				t.Errorf("Handler returned unexpected body: got '%s', want '%s'",
					strings.TrimSpace(rr.Body.String()), test.expectedBody)
			}

			if test.checkHeader {
				contentType := rr.Header().Get("Content-Type")
				expectedContentType := "text/plain"
				assert.Equal(t, expectedContentType, contentType,
					"Handler returned wrong content type: got %s, want %s", contentType, expectedContentType)
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
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.RedirectURL(rr, req)

			assert.Equal(t, test.expectedStatus, rr.Code,
				"Handler returned unexpected status: got %d, want %d", rr.Code, test.expectedStatus)

			if test.expectedLocation != "" {
				location := rr.Header().Get("Location")
				assert.Equal(t, test.expectedLocation, location,
					"Handler returned wrong location in header: got %s, want %s", location, test.expectedLocation)
			}

			body := strings.TrimSpace(rr.Body.String())
			if test.expectedBody != "" {
				assert.Equal(t, test.expectedBody, body,
					"Handler returned unexpected body: got '%s', want '%s'", body, test.expectedBody)
			}
		})
	}
}

func TestCreateShortURLJson(t *testing.T) {
	// создали хранилище
	mockStorage := NewMockStorage()
	// создали handler
	handler := NewHandler(mockStorage, config.HOST)

	tests := []struct {
		name       string
		request    model.ShortURLJSON
		userID     string
		wantStatus int
		wantError  bool
	}{
		{
			name:       "успешный запрос",
			request:    model.ShortURLJSON{URL: "https://yandex.ru"},
			userID:     "test-user-1",
			wantStatus: http.StatusCreated,
			wantError:  false,
		},
		{
			name:       "пустой URL",
			request:    model.ShortURLJSON{URL: ""},
			userID:     "test-user-1",
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "URL с пробелами",
			request:    model.ShortURLJSON{URL: "   "},
			userID:     "test-user-1",
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "без user ID",
			request:    model.ShortURLJSON{URL: "https://yandex.ru"},
			userID:     "",
			wantStatus: http.StatusUnauthorized,
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

			// Добавляем userID в контекст если он указан
			if test.userID != "" {
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, test.userID)
				req = req.WithContext(ctx)
			}

			// записываем всё это в recover
			res := httptest.NewRecorder()
			// выполняем функцию с нашими данными
			handler.CreateShortURLJson(res, req)

			// проверяем по ошибкам
			assert.Equal(t, test.wantStatus, res.Code,
				"Status got: %d, want: %d", res.Code, test.wantStatus)

			if !test.wantError {
				contentType := res.Header().Get("Content-Type")
				assert.Equal(t, "application/json", contentType,
					"Content-Type should be application/json")

				if res.Code == test.wantStatus {
					var response model.ShortURLJSONResult
					err := json.Unmarshal(res.Body.Bytes(), &response)
					assert.NoError(t, err, "JSON should be valid")
					assert.NotEmpty(t, response.Result, "Response result should not be empty")
				}
			}
		})
	}
}

func TestGetUserURLs(t *testing.T) {
	mockStorage := NewMockStorage()
	handler := NewHandler(mockStorage, config.HOST)

	tests := []struct {
		name           string
		userID         string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "User with URLs",
			userID:         "user",
			expectedStatus: http.StatusOK,
			expectedCount:  1, // У пользователя 'user' есть 1 URL
		},
		{
			name:           "Another user with URLs",
			userID:         "test",
			expectedStatus: http.StatusOK,
			expectedCount:  1, // У пользователя 'test' есть 1 URL
		},
		{
			name:           "User without URLs",
			userID:         "unknown-user",
			expectedStatus: http.StatusNoContent,
			expectedCount:  0,
		},
		{
			name:           "No user ID",
			userID:         "",
			expectedStatus: http.StatusUnauthorized,
			expectedCount:  0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/user/urls", nil)

			// Добавляем userID в контекст если он указан
			if test.userID != "" {
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, test.userID)
				req = req.WithContext(ctx)
			}

			rr := httptest.NewRecorder()
			handler.GetURLByUser(rr, req)

			assert.Equal(t, test.expectedStatus, rr.Code,
				"Expected status %d, got %d", test.expectedStatus, rr.Code)

			if test.expectedStatus == http.StatusOK {
				var urls []storage.OriginalAndShortURLs
				err := json.Unmarshal(rr.Body.Bytes(), &urls)
				assert.NoError(t, err, "Should return valid JSON")
				assert.Len(t, urls, test.expectedCount,
					"Expected %d URLs, got %d", test.expectedCount, len(urls))
			}
		})
	}
}

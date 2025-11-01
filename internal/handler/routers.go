package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/config/db"
	"github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/middleware"
	"github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/model"
	"github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/storage"
)

type Handler struct {
	storage storage.URLStorage
	baseURL string
}

func NewHandler(storage storage.URLStorage, baseURL string) *Handler {
	return &Handler{
		storage: storage,
		baseURL: baseURL,
	}
}

func (h *Handler) CreateShortURLJson(w http.ResponseWriter, r *http.Request) {
	var shortURLJSON model.ShortURLJSON
	var shortURLJSONResult model.ShortURLJSONResult

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&shortURLJSON); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(shortURLJSON.URL) == "" {
		http.Error(w, "URL not be empty", http.StatusBadRequest)
		return
	}

	// получаем userID из контекста
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	shortID, exists := h.storage.Save(shortURLJSON.URL, userID)
	shortURL := h.baseURL + "/" + shortID

	shortURLJSONResult.Result = shortURL

	w.Header().Set("Content-Type", "application/json")
	if exists {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(shortURLJSONResult)

}

func (h *Handler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	originalURL := string(body)
	if originalURL == "" {
		http.Error(w, "URL not be empty", http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	shortID, exists := h.storage.Save(originalURL, userID)
	shortUIL := h.baseURL + "/" + shortID

	w.Header().Set("Content-Type", "text/plain")
	if exists {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	w.Write([]byte(shortUIL))
}

func (h *Handler) RedirectURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/")

	if id == "" {
		http.Error(w, "ID cannot be empty", http.StatusBadRequest)
		return
	}

	originalURL, exists := h.storage.Get(id)
	if !exists {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *Handler) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Invalid request", http.StatusBadRequest)
}

func (h *Handler) MethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *Handler) PingPostgres(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(); err != nil {
		http.Error(w, "postgres database connection error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *Handler) CreateShortURLBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var batchRequests []model.BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&batchRequests); err != nil {
		http.Error(w, "error decode json body", http.StatusBadRequest)
		return
	}

	if len(batchRequests) == 0 {
		http.Error(w, "body can't be empty", http.StatusBadRequest)
		return
	}

	// Проверяем все URL перед обработкой
	for _, req := range batchRequests {
		if strings.TrimSpace(req.OriginalURL) == "" {
			http.Error(w, "url can't be empty", http.StatusBadRequest)
			return
		}
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Используем батч сохранение если доступно
	if batchStorage, ok := h.storage.(interface {
		SaveBatch(urls []string, userID string) []string
	}); ok {

		// Для PostgreSQL - батчевое сохранение в транзакции
		urls := make([]string, len(batchRequests))
		for i, req := range batchRequests {
			urls[i] = req.OriginalURL
		}

		shortIDs := batchStorage.SaveBatch(urls, userID)
		batchResponse := make([]model.BatchResponse, len(batchRequests))

		for i, req := range batchRequests {
			batchResponse[i] = model.BatchResponse{
				CorrelationID: req.CorrelationID,
				ShortURL:      h.baseURL + "/" + shortIDs[i],
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(batchResponse)
		return
	}

	// Fallback - последовательное сохранение для файлового хранилища
	batchResponse := make([]model.BatchResponse, 0, len(batchRequests))
	for _, req := range batchRequests {
		shortID, _ := h.storage.Save(req.OriginalURL, userID)
		shortURL := h.baseURL + "/" + shortID

		batchResponse = append(batchResponse, model.BatchResponse{
			CorrelationID: req.CorrelationID,
			ShortURL:      shortURL,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(batchResponse)
}

func (h *Handler) GetURLByUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	urls, err := h.storage.GetURLByUser(userID)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	for i := range urls {
		if !strings.Contains(urls[i].ShortURL, "://") {
			urls[i].ShortURL = h.baseURL + "/" + urls[i].ShortURL
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(urls)
}

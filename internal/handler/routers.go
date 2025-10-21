package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/config/db"
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
	var shortURLJSON model.ShortUrlJson
	var shortURLJSONResult model.ShortUrlJsonResult

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
	}

	shortID := h.storage.Save(shortURLJSON.URL)
	shortURL := h.baseURL + "/" + shortID

	shortURLJSONResult.Result = shortURL

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
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

	shortID := h.storage.Save(originalURL)
	shortUIL := h.baseURL + "/" + shortID

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
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

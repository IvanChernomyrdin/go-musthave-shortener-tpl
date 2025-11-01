package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/model"
)

type URLRecord struct {
	OriginalURL string
	UserID      string
}

type OriginalAndShortURLs struct {
	OriginalURL string
	ShortURL    string
}

type FileStorage struct {
	mu       sync.RWMutex
	filepath string
	urls     map[string]URLRecord
	counter  int64
	baseURL  string
}

func NewFileStorage(filepath, baseURL string) (*FileStorage, error) {
	storage := &FileStorage{
		filepath: filepath,
		urls:     make(map[string]URLRecord),
		counter:  0,
		baseURL:  baseURL,
	}
	if err := storage.loadFromFile(); err != nil {
		return nil, err
	}
	return storage, nil
}

func (fs *FileStorage) loadFromFile() error {
	var storages []model.URLStorage

	if _, err := os.Stat(fs.filepath); os.IsNotExist(err) {
		return nil
	}
	data, err := os.ReadFile(fs.filepath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &storages); err != nil {
		return err
	}
	fs.urls = make(map[string]URLRecord)
	for _, storage := range storages {
		fs.urls[storage.ID] = URLRecord{
			OriginalURL: storage.OriginalURL,
			UserID:      storage.UserID,
		}

		if id, err := strconv.ParseInt(storage.ID, 10, 64); err == nil && id > fs.counter {
			fs.counter = id
		}
	}
	return nil
}

func (fs *FileStorage) Get(id string) (string, bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	url, exists := fs.urls[id]
	return url.OriginalURL, exists
}

func (fs *FileStorage) Save(originalURL, userID string) (string, bool) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Сначала проверяем, нет ли уже такого URL
	for id, url := range fs.urls {
		if url.OriginalURL == originalURL {
			return id, true // конфликт - URL уже существует
		}
	}

	// Если это новый URL - создаем новую запись
	fs.counter++
	id := strconv.FormatInt(fs.counter, 10)

	fs.urls[id] = URLRecord{
		OriginalURL: originalURL,
		UserID:      userID,
	}

	// Сохраняем в файл (вызываем существующий метод)
	if err := fs.SaveToFile(); err != nil {
		return id, false
	}

	return id, false
}

func (fs *FileStorage) SaveToFile() error {
	var storage []model.URLStorage

	for id, record := range fs.urls {
		storage = append(storage, model.URLStorage{
			ID:          id,
			ShortURL:    fs.baseURL + "/" + id,
			OriginalURL: record.OriginalURL,
			UserID:      record.UserID,
		})
	}
	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return err
	}

	// Создаем директорию если её нет
	dir := filepath.Dir(fs.filepath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(fs.filepath, data, 0644)
}

func (fs *FileStorage) GetURLByUser(user string) ([]OriginalAndShortURLs, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var result []OriginalAndShortURLs

	for id, record := range fs.urls {
		if record.UserID == user {
			result = append(result, OriginalAndShortURLs{
				OriginalURL: record.OriginalURL,
				ShortURL:    fs.baseURL + "/" + id,
			})
		}
	}

	return result, nil
}

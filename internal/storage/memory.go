package storage

import (
	"strconv"
	"sync"
)

type URLStorage interface {
	Save(url, userid string) (string, bool)
	Get(id string) (string, bool)
	GetURLByUser(userID string) ([]OriginalAndShortURLs, error)
	DeleteURLs(userID string, urlIDs []string) error
}
type MemoryStorage struct {
	mu      sync.RWMutex
	url     map[string]URLRecord
	counter int64
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		url:     make(map[string]URLRecord),
		counter: 0,
	}
}

func (ms *MemoryStorage) Save(originalURL, userID string) (string, bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Сначала проверяем, нет ли уже такого URL
	for id, url := range ms.url {
		if url.OriginalURL == originalURL {
			return id, true // конфликт - URL уже существует
		}
	}

	// Если это новый URL - создаем новую запись
	ms.counter++
	id := strconv.Itoa(int(ms.counter))
	ms.url[id] = URLRecord{
		OriginalURL: originalURL,
		UserID:      userID,
	}

	return id, false // нет конфликта
}

func (ms *MemoryStorage) Get(id string) (string, bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	url, exists := ms.url[id]
	return url.OriginalURL, exists
}

func (ms *MemoryStorage) GetURLByUser(userID string) ([]OriginalAndShortURLs, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var urls []OriginalAndShortURLs
	for id, record := range ms.url {
		if record.UserID == userID {
			urls = append(urls, OriginalAndShortURLs{
				OriginalURL: record.OriginalURL,
				ShortURL:    id,
			})
		}
	}
	return urls, nil
}

func (ms *MemoryStorage) DeleteURLs(userID string, urlIDs []string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, id := range urlIDs {
		if record, exists := ms.url[id]; exists && record.UserID == userID {
			// Помечаем как удаленное (или удаляем из мапы)
			record.IsDeleted = true
			ms.url[id] = record
		}
	}
	return nil
}

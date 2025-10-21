package storage

import (
	"strconv"
	"sync"
)

type URLStorage interface {
	Save(url string) (string, bool)
	Get(id string) (string, bool)
}
type MemoryStorage struct {
	mu      sync.RWMutex
	url     map[string]string
	counter int64
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		url:     make(map[string]string),
		counter: 0,
	}
}

func (ms *MemoryStorage) Save(originalURL string) (string, bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Сначала проверяем, нет ли уже такого URL
	for id, url := range ms.url {
		if url == originalURL {
			return id, true // конфликт - URL уже существует
		}
	}

	// Если это новый URL - создаем новую запись
	ms.counter++
	id := strconv.Itoa(int(ms.counter))
	ms.url[id] = originalURL

	return id, false // нет конфликта
}

func (ms *MemoryStorage) Get(id string) (string, bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	url, exists := ms.url[id]
	return url, exists
}

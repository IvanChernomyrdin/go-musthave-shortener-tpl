package storage

import (
	"strconv"
	"sync"
)

type URLStorage interface {
	Save(url string) string
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

func (ms *MemoryStorage) Save(url string) string {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.counter++
	id := strconv.Itoa(int(ms.counter))

	ms.url[id] = url

	return id
}

func (ms *MemoryStorage) Get(id string) (string, bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	url, exists := ms.url[id]
	return url, exists
}

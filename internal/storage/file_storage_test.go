package storage

import (
	"path/filepath"
	"testing"

	"github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/config"
)

func TestGetAndSave(t *testing.T) {
	cfg := config.NewConfig()
	tmpFile := filepath.Join(t.TempDir(), "test_json.json")

	//создаём новое хранилище
	storage, err := NewFileStorage(tmpFile, cfg.BaseURL)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}
	originalURL := "https://yandex.ru"
	// сохраняем
	shortID := storage.Save(originalURL)

	// должны получить ID
	if shortID == "" {
		t.Error("Save should return non-empty short ID")
	}
	// по id получаем url
	url, exists := storage.Get(shortID)
	if !exists {
		t.Fatal("URL should exists after saving")
	}
	if url != originalURL {
		t.Errorf("Expected URL:%s, got: %s", originalURL, url)
	}
	// получаем url несуществующей
	_, exists = storage.Get("88005553535")
	if exists {
		t.Fatal("Not existent url should not exists")
	}

}

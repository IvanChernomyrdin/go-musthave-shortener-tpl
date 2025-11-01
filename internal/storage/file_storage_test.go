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
	userID := "test-user-123"
	// сохраняем
	shortID, _ := storage.Save(originalURL, userID)

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

	userURLs, err := storage.GetURLByUser(userID)
	if err != nil {
		t.Errorf("userURLs got %v", err)
	}

	if len(userURLs) != 1 {
		t.Errorf("Expected 1 URL for user (because we saved only one), got %d", len(userURLs))
	}

	if userURLs[0].OriginalURL != originalURL {
		t.Errorf("Expected url: %s, got: %s", originalURL, userURLs[0].OriginalURL)
	}

	otherUserURLs, err := storage.GetURLByUser("jopa-test")

	if err != nil {
		t.Errorf("otherUserURLs got %v", err)
	}

	if len(otherUserURLs) != 0 {
		t.Errorf("Other user should have 0 URLs, got %d", len(otherUserURLs))
	}
}

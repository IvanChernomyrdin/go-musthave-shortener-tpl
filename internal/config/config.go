package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	ServerAddress string
	BaseURL       string
	Port          string
	FileStorage   string
	DatabaseDSN   string
}

const HOST = "http://localhost:8080"
const EncryptionKey = "32-bytes-long-key-1234567890777!"

func NewConfig() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "HTTP server address (host:port)")
	flag.StringVar(&cfg.BaseURL, "b", HOST, "Base URL for shortened links")
	flag.StringVar(&cfg.FileStorage, "f", "internal/config/storage/file_storage.json", "File name where the data will be stored shorts url")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "Connection string databse postgres")

	flag.Parse()

	if envAddr := os.Getenv("SERVER_ADDRESS"); envAddr != "" {
		cfg.ServerAddress = envAddr
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}
	if envFileStorage := os.Getenv("FILE_STORAGE_PATH"); envFileStorage != "" {
		cfg.FileStorage = envFileStorage
	}
	if envDatabaseDSN := os.Getenv("DATABASE_DSN"); envDatabaseDSN != "" {
		cfg.DatabaseDSN = envDatabaseDSN
	}

	cfg.Port = cfg.extractPort()

	return cfg
}

func (c *Config) extractPort() string {
	colonIndex := strings.LastIndex(c.ServerAddress, ":")
	if colonIndex != -1 {
		return c.ServerAddress[colonIndex:]
	}
	return ":8080"
}

func (c *Config) Validate() error {
	if c.ServerAddress == "" {
		return fmt.Errorf("server address cannot be empty")
	}
	if c.BaseURL == "" {
		return fmt.Errorf("base URL cannot be empty")
	}
	return nil
}

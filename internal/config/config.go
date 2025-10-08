package config

import (
	"flag"
	"fmt"
	"strings"
)

type Config struct {
	ServerAddress string
	BaseURL       string
	Port          string
}

const HOST = "http://localhost:8080"

func NewConfig() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "HTTP server address (host:port)")
	flag.StringVar(&cfg.BaseURL, "b", HOST, "Base URL for shortened links")

	flag.Parse()

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

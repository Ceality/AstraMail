// Package config provides configuration parsing and environment management for AstraMail.
package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	// defaultServerHost 是未配置 SERVER_HOST 时的监听地址。
	defaultServerHost = "127.0.0.1"
	// defaultServerPort 是未配置 SERVER_PORT 时的监听端口。
	defaultServerPort uint16 = 3000
)

// Config represents application configuration parameters.
type Config struct {
	ServerHost string
	ServerPort uint16
}

// Load loads configuration from environment variables and .env file.
func Load() *Config {
	// Attempt to load .env from current directory or parent directory
	err := godotenv.Load()
	if err != nil {
		errParent := godotenv.Load(".env")
		if errParent != nil {
			log.Println("No .env file found, using system environment variables or defaults")
		}
	}

	host := os.Getenv("SERVER_HOST")
	if host == "" {
		host = defaultServerHost
	}

	port := defaultServerPort

	if portStr := os.Getenv("SERVER_PORT"); portStr != "" {
		parsed, parseErr := strconv.ParseUint(portStr, 10, 16)
		if parseErr == nil {
			port = uint16(parsed)
		} else {
			log.Printf("Invalid SERVER_PORT, falling back to %d", port)
		}
	}

	return &Config{
		ServerHost: host,
		ServerPort: port,
	}
}

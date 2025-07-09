package config

import (
	"log"
	"os"
)

// Config holds the application configuration.
type Config struct {
	Websocket Websocket
}

// Websocket holds the Websocket configuration.
type Websocket struct {
	Port string
}

// LoadConfig loads the configuration from environment variables.
func LoadConfig() *Config {
	config := &Config{}

	config.Websocket.Port = os.Getenv("WEBSOCKET_PORT")
	if config.Websocket.Port == "" {
		log.Fatal("WEBSOCKET_PORT environment variable is not set")
	}

	return config
}

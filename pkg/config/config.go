package config

import (
	"os"
)

type Config struct {
	BrokerAdress string
	ClientID     string
	DatabaseURL  string
}

func newConfig() *Config {
	return &Config{
		BrokerAdress: getEnv("MQTT_BROKER_ADDRESS", "tcp://localhost:1883"),
		ClientID:     getEnv("MQTT_CLIENT_ID", "chat-server"),
		DatabaseURL:  getEnv("DATABASE_URL", "mongodb://localhost:27017"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

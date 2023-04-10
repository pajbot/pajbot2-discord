package config

import (
	"os"
	"strings"
)

func mustStringEnv(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	panic("Missing required environment variable: " + key)
}

func stringEnv(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return defaultValue
}

func stringListEnv(key string, defaultValue []string) []string {
	if value, ok := os.LookupEnv(key); ok {
		return strings.Split(value, ",")
	}

	return defaultValue
}

package config

import (
	"os"
	"strconv"
	"time"
)

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvAsInt(key string, fallback int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return fallback
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return fallback
	}
	return value
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return fallback
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return fallback
	}
	return value
}

package config

import (
	"fmt"
	"time"
)

type DBConfig struct {
	Connection      string
	Host            string
	Port            int
	Database        string
	Username        string
	Password        string
	SSLMode         string
	Timezone        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func LoadDBConfig() DBConfig {
	return DBConfig{
		Connection: getEnv("DB_CONNECTION", "postgres"),
		Host:       getEnv("DB_HOST", "127.0.0.1"),
		Port:       getEnvAsInt("DB_PORT", 5432),
		Database:   getEnv("DB_DATABASE", "clinic_db"),
		Username:   getEnv("DB_USERNAME", "postgres"),
		Password:   getEnv("DB_PASSWORD", ""),
		SSLMode:    getEnv("DB_SSLMODE", "disable"),
		Timezone:   getEnv("DB_TIMEZONE", "UTC"),
	}
}

func (c DBConfig) DSN() string {
	switch c.Connection {
	case "postgres":
		return fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
			c.Host,
			c.Port,
			c.Username,
			c.Password,
			c.Database,
			c.SSLMode,
			c.Timezone,
		)
	default:
		return ""
	}
}

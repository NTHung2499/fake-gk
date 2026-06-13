package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port  string
	MySQL MySQLConfig
}

type MySQLConfig struct {
	Host            string
	Port            string
	Database        string
	User            string
	Password        string
	ConnectionLimit int
}

func Load() Config {
	return Config{
		Port: env("PORT", "3000"),
		MySQL: MySQLConfig{
			Host:            env("MYSQL_HOST", "mysql.database.svc.cluster.local"),
			Port:            env("MYSQL_PORT", "3306"),
			Database:        env("MYSQL_DATABASE", "appdb"),
			User:            env("MYSQL_USER", "appuser"),
			Password:        env("MYSQL_PASSWORD", "apppass123"),
			ConnectionLimit: envInt("MYSQL_CONNECTION_LIMIT", 10),
		},
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

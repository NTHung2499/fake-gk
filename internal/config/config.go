package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port   string
	MySQL  MySQLConfig
	App    AppConfig
	OpenAI OpenAIConfig
}

type MySQLConfig struct {
	Host            string
	Port            string
	Database        string
	User            string
	Password        string
	ConnectionLimit int
}

type AppConfig struct {
	Secret string
}

type OpenAIConfig struct {
	Model                 string
	ContextMessages       int
	RequestTimeoutSeconds int
}

func Load() Config {
	appSecret := os.Getenv("APP_SECRET")
	if appSecret == "" && os.Getenv("GIN_MODE") != "release" {
		appSecret = "dev-only-fakegk-secret-change-me"
	}

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
		App: AppConfig{
			Secret: appSecret,
		},
		OpenAI: OpenAIConfig{
			Model:                 env("OPENAI_MODEL", "gpt-5.4-mini"),
			ContextMessages:       envInt("CHAT_CONTEXT_MESSAGES", 30),
			RequestTimeoutSeconds: envInt("OPENAI_REQUEST_TIMEOUT_SECONDS", 60),
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

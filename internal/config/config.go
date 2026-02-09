// config/config.go
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	RabbitMQ RabbitMQConfig
	Email    EmailConfig
	Telegram TelegramConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type RabbitMQConfig struct {
	Host      string
	Port      string
	User      string
	Password  string
	QueueName string
}

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	FromEmail    string
	FromName     string
	AuthUser     string
	AuthPassword string
}

type TelegramConfig struct {
	BotToken string
	APIURL   string
}

func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "delayed_notifier"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		RabbitMQ: RabbitMQConfig{
			Host:      getEnv("RABBITMQ_HOST", "localhost"),
			Port:      getEnv("RABBITMQ_PORT", "5672"),
			User:      getEnv("RABBITMQ_USER", "guest"),
			Password:  getEnv("RABBITMQ_PASSWORD", "guest"),
			QueueName: getEnv("RABBITMQ_QUEUE", "notifications"),
		},
		Email: EmailConfig{
			SMTPHost:     getEnv("EMAIL_SMTP_HOST", "smtp.gmail.com"),
			SMTPPort:     getIntEnv("EMAIL_SMTP_PORT", 587),
			FromEmail:    getEnv("EMAIL_FROM", ""),
			FromName:     getEnv("EMAIL_FROM_NAME", "DelayedNotifier"),
			AuthUser:     getEnv("EMAIL_AUTH_USER", ""),
			AuthPassword: getEnv("EMAIL_AUTH_PASSWORD", ""),
		},
		Telegram: TelegramConfig{
			BotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
			APIURL:   getEnv("TELEGRAM_API_URL", "https://api.telegram.org/bot"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

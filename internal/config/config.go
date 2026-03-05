// config/config.go
package config

import (
	"github.com/cockroachdb/errors"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Addr        string      `mapstructure:"addr"`
	DatabaseURL string      `mapstructure:"db_url"`
	RedisURL    string      `mapstructure:"redis_url"`
	RabbitMQURL string      `mapstructure:"rabbitmq_url"`
	BotToken    string      `mapstructure:"bot_token"`
	Email       EmailConfig `mapstructure:"email"`
}

type EmailConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
}

func Load() (Config, error) {
	_ = godotenv.Load()

	v := viper.New()
	v.AddConfigPath(".")
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	v.BindEnv("addr", "APP_ADDR")
	v.BindEnv("db_url", "DB_URL")
	v.BindEnv("redis_url", "REDIS_URL")
	v.BindEnv("rabbitmq_url", "RABBITMQ_URL")
	v.BindEnv("bot_token", "BOT_TOKEN")

	// Для вложенных полей используем точку
	v.BindEnv("email.host", "EMAIL_HOST")
	v.BindEnv("email.port", "EMAIL_PORT")
	v.BindEnv("email.user", "EMAIL_USER")
	v.BindEnv("email.password", "EMAIL_PASSWORD")
	v.BindEnv("email.from", "EMAIL_FROM")

	v.AutomaticEnv()

	err := v.ReadInConfig()
	if err != nil {
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return Config{}, errors.WithStack(err)
		}
	}
	
	var cfg Config
	if err = v.Unmarshal(&cfg); err != nil {
		return Config{}, errors.WithDetail(err, "unable to decode into struct")
	}

	return cfg, nil
}

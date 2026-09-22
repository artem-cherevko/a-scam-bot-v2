package config

import (
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	API_PORT      string `env:"API_PORT" envDefault:"8080"`
	DSN           string `env:"DB_DSN" envDefault:"postgres://postgres:postgres@postgres:5432/a_scam?sslmode=disable"`
	BOT_TOKEN     string `env:"BOT_TOKEN,required"`
	BOT_USER_NAME string `env:"BOT_USER_NAME,required"`

	ChannelID int64 `env:"PROJECT_CHANNEL_ID,required"`

	SUBSCRIPTIONS_CHAT_ID int64 `env:"SUBSCRIPTIONS_CHAT_ID"`

	// REQUIRED URLs
	PROJECT_CHANEL_URL   string `env:"PROJECT_CHANEL_URL,required"`
	SUBMISSIONS_CHAT_URL string `env:"SUBMISSIONS_CHAT_URL,required"`
	PROJECT_CHAT_URL     string `env:"PROJECT_CHAT_URL,required"`
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = env.Parse(&cfg)

	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

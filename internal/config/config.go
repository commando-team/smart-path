package config

import (
	"log"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

var (
	EnvDevelopment = "dev"
	EnvRelease     = "release"
)

type Config struct {
	Environment string `env:"ENV" env-default:"dev"`

	DBUser     string `env:"DB_USER" env-required:"true"`
	DBPassword string `env:"DB_PASSWORD" env-required:"true"`
	DBHost     string `env:"DB_HOST" env-required:"true"`
	DBPort     string `env:"DB_PORT" env-default:"5432"`
	DBName     string `env:"DB_NAME" env-required:"true"`

	Port string `env:"SERVER_PORT" env-default:"8080"`
}

func MustLoad() *Config {
	var cfg Config

	godotenv.Load()

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("failed to load config: %s", err)
	}

	return &cfg
}

package config

import (
	"os"

	"github.com/devmin8/rivet/internal/validation"
	"github.com/joho/godotenv"
)

type ServerEnv struct {
	Port   string `validate:"required"`
	Domain string `validate:"required,domain_or_url"`
	DBPath string `validate:"required"`
}

var validate = validation.New()

func Load() (*ServerEnv, error) {
	env := os.Getenv("GO_ENV")
	if env != "production" {
		godotenv.Load()
	}

	cfg := getConfig()
	if err := validate.Struct(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getConfig() *ServerEnv {
	return &ServerEnv{
		Port:   os.Getenv("PORT"),
		Domain: os.Getenv("DOMAIN"),
		DBPath: os.Getenv("DB_PATH"),
	}
}

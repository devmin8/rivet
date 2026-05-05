package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/devmin8/rivet/internal/validation"
	"github.com/joho/godotenv"
)

type AppEnv string

const (
	Dev  AppEnv = "dev"
	Prod AppEnv = "prod"
)

type ServerEnv struct {
	Port     int    `validate:"required"`
	Domain   string `validate:"required,domain_or_url"`
	DBPath   string `validate:"required"`
	CaddyURL string `validate:"required,url"`
	AppEnv   AppEnv `validate:"required,oneof=dev prod"`
}

var validate = validation.New()

func Load() (*ServerEnv, error) {
	if os.Getenv("GO_ENV") != "production" {
		_ = godotenv.Load()
	}

	cfg, err := getConfig()
	if err != nil {
		return nil, err
	}

	if err := validate.Struct(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getConfig() (*ServerEnv, error) {
	port, err := envInt("PORT")
	if err != nil {
		return nil, err
	}

	return &ServerEnv{
		Port:     port,
		Domain:   env("DOMAIN"),
		DBPath:   env("DB_PATH"),
		CaddyURL: envWithDefault("CADDY_URL", "http://rivet-caddy:2019"),
		AppEnv:   AppEnv(envWithDefault("APP_ENV", "dev")),
	}, nil
}

func env(key string) string {
	return os.Getenv(key)
}

func envInt(key string) (int, error) {
	val := env(key)
	if val == "" {
		return 0, fmt.Errorf("%s is required", key)
	}

	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}

	return n, nil
}

func envWithDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

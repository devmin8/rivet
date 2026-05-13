package config

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/devmin8/rivet/internal/validation"
	"github.com/joho/godotenv"
)

const secretKeyEnvName = "RIVET_SECRET_KEY"

type AppEnv string

const (
	Dev  AppEnv = "dev"
	Prod AppEnv = "prod"
)

type ServerEnv struct {
	Port               int    `validate:"required"`
	Domain             string `validate:"required,domain_or_url"`
	DBPath             string `validate:"required"`
	CaddyURL           string `validate:"required,url"`
	CaddyAccessLogPath string `validate:"required"`
	AppEnv             AppEnv `validate:"required,oneof=dev prod"`
	SecretKey          []byte `validate:"required,len=32"`
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

	secretKey, err := envSecretKey()
	if err != nil {
		return nil, err
	}

	return &ServerEnv{
		Port:               port,
		Domain:             env("DOMAIN"),
		DBPath:             env("DB_PATH"),
		CaddyURL:           envWithDefault("CADDY_URL", "http://rivet-caddy:2019"),
		CaddyAccessLogPath: envWithDefault("CADDY_ACCESS_LOG_PATH", "/var/log/rivet-caddy/access.log"),
		AppEnv:             AppEnv(envWithDefault("APP_ENV", "dev")),
		SecretKey:          secretKey,
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

func envSecretKey() ([]byte, error) {
	value := env(secretKeyEnvName)
	if value == "" {
		return nil, errors.New("RIVET_SECRET_KEY is required")
	}

	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil && len(decoded) == 32 {
		return decoded, nil
	}

	if decoded, err := hex.DecodeString(value); err == nil && len(decoded) == 32 {
		return decoded, nil
	}

	return nil, errors.New("RIVET_SECRET_KEY must be a 64-character hex string or base64-encoded 32-byte key")
}

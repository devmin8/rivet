package web

import (
	"log/slog"

	"github.com/devmin8/rivet/internal/docker"
	"github.com/devmin8/rivet/internal/server/config"
	"github.com/devmin8/rivet/internal/validation"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type WebContext struct {
	cfg    *config.ServerEnv
	db     *gorm.DB
	log    *slog.Logger
	docker *docker.Client
}

func NewServer(cfg *config.ServerEnv, db *gorm.DB, dockerClient *docker.Client, log *slog.Logger) *fiber.App {
	webCtx := &WebContext{cfg: cfg, db: db, log: log, docker: dockerClient}

	app := fiber.New(fiber.Config{
		StructValidator:              validation.New(),
		StreamRequestBody:            true,
		DisablePreParseMultipartForm: true,
		// Allow large Docker image tarballs to stream through app image upload.
		// 2.15 GB
		BodyLimit: 2 * 1024 * 1024 * 1024,
	})

	return registerRoutes(app, webCtx)
}

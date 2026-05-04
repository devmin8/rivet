package web

import (
	"log/slog"

	"github.com/devmin8/rivet/internal/server/config"
	"github.com/devmin8/rivet/internal/validation"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type WebContext struct {
	cfg *config.ServerEnv
	db  *gorm.DB
	log *slog.Logger
}

func NewServer(cfg *config.ServerEnv, db *gorm.DB, log *slog.Logger) *fiber.App {
	webCtx := &WebContext{cfg, db, log}

	app := fiber.New(fiber.Config{
		StructValidator: validation.New(),
	})

	return registerRoutes(app, webCtx)
}

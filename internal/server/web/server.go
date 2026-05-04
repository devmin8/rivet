package web

import (
	"log/slog"

	"github.com/devmin8/rivet/internal/server/config"
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
	return registerRoutes(fiber.New(), webCtx)
}

package web

import (
	"log/slog"

	"github.com/devmin8/rivet/internal/server/config"
	"github.com/gofiber/fiber/v3"
)

func NewServer(cfg *config.ServerEnv, logger *slog.Logger) error {
	app := fiber.New()

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	registerRoutes(app)

	logger.Info("server started", "port", cfg.Port)
	return app.Listen(":" + cfg.Port)
}

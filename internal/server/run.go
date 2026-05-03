package server

import (
	"github.com/devmin8/rivet/internal/logger"
	"github.com/devmin8/rivet/internal/server/config"
	"github.com/gofiber/fiber/v3"
)

func Run() error {
	logger := logger.NewLogger()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		return err
	}

	// Fiber server setup
	app := fiber.New()

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	return app.Listen(":" + cfg.Port)
}

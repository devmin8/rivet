package web

import (
	"github.com/devmin8/rivet/internal/server/services"
	"github.com/devmin8/rivet/internal/server/web/handlers"
	"github.com/gofiber/fiber/v3"
)

func registerRoutes(app *fiber.App, webCtx *WebContext) *fiber.App {
	// Root route for health check
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// API routes
	api := app.Group("/api")
	v1 := api.Group("/v1")

	v1.Get("/health", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	authService := services.NewAuthService(webCtx.db, webCtx.log)
	authHandler := handlers.NewAuthHandler(authService)
	v1.Post("/auth/register", authHandler.RegisterUser)

	return app
}

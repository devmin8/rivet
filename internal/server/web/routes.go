package web

import (
	"github.com/devmin8/rivet/internal/server/services"
	"github.com/devmin8/rivet/internal/server/web/handlers"
	"github.com/devmin8/rivet/internal/server/web/middlewares"
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

	authService := services.NewAuthService(webCtx.db, webCtx.log)
	authHandler := handlers.NewAuthHandler(authService)
	v1.Post("/auth/register", authHandler.RegisterUser)
	v1.Post("/auth/signin", authHandler.SignInUser)

	v1.Use(middlewares.RequireAuth(authService))

	v1.Get("/health", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	projectService := services.NewProjectService(webCtx.db)
	projectHandler := handlers.NewProjectHandler(projectService)
	v1.Post("/projects", projectHandler.CreateProject)

	return app
}

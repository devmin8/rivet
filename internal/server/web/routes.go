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

	v1.Get("/health", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	authService := services.NewAuthService(webCtx.db, webCtx.log)
	authHandler := handlers.NewAuthHandler(authService)
	v1.Post("/auth/register", authHandler.RegisterUser)
	v1.Post("/auth/signin", authHandler.SignInUser)

	v1.Use(middlewares.RequireAuth(authService))
	v1.Use(middlewares.RequireCSRF())

	v1.Get("/auth/me", authHandler.CurrentUser)

	projectService := services.NewProjectServiceWithLogger(webCtx.db, webCtx.docker, webCtx.log)
	projectHandler := handlers.NewProjectHandler(projectService, webCtx.routes, webCtx.log)
	v1.Get("/projects/stats", projectHandler.GetProjectStats)
	v1.Get("/projects", projectHandler.ListProjects)
	v1.Post("/projects", projectHandler.CreateProject)
	v1.Get("/projects/:id", projectHandler.GetProject)
	v1.Patch("/projects/:id/runtime-settings", projectHandler.UpdateProjectRuntimeSettings)
	v1.Post("/projects/:id/start", projectHandler.StartProject)
	v1.Post("/projects/:id/stop", projectHandler.StopProject)
	v1.Delete("/projects/:id", projectHandler.DeleteProject)
	v1.Post("/projects/:id/deploy", projectHandler.DeployProject)

	imageHandler := handlers.NewImageHandler(projectService, webCtx.docker)
	v1.Post("/projects/:id/images/upload", imageHandler.UploadImage)

	return app
}

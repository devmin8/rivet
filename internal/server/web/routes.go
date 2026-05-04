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
	v1.Get("/projects", projectHandler.ListProjects)
	v1.Get("/projects/:id", projectHandler.GetProject)
	v1.Delete("/projects/:id", projectHandler.DeleteProject)

	appService := services.NewAppService(webCtx.db)
	appHandler := handlers.NewAppHandler(appService)
	v1.Post("/projects/:projectID/apps", appHandler.CreateApp)
	v1.Get("/projects/:projectID/apps", appHandler.ListApps)

	v1.Get("/apps/:appID", appHandler.GetApp)
	v1.Delete("/apps/:appID", appHandler.DeleteApp)

	imageHandler := handlers.NewImageHandler(appService, webCtx.docker)
	v1.Post("/apps/:appID/images/upload", imageHandler.UploadImage)

	deploymentService := services.NewDeploymentService(webCtx.db, webCtx.docker)
	deploymentHandler := handlers.NewDeploymentHandler(deploymentService)
	v1.Post("/apps/:appID/deployments", deploymentHandler.CreateDeployment)
	v1.Get("/apps/:appID/deployments", deploymentHandler.ListDeployments)
	v1.Post("/apps/:appID/start", deploymentHandler.StartAppContainer)
	v1.Post("/apps/:appID/stop", deploymentHandler.StopAppContainer)
	v1.Delete("/apps/:appID/container", deploymentHandler.DeleteAppContainer)

	return app
}

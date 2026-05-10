package handlers

import (
	"context"
	"errors"
	"html"
	"log/slog"
	"strings"

	"github.com/devmin8/rivet/internal/server/database"
	"github.com/devmin8/rivet/internal/server/services"
	"github.com/gofiber/fiber/v3"
)

type WakeHandler struct {
	projects     *services.ProjectService
	routes       RouteSyncer
	serverDomain string
	log          *slog.Logger
}

func NewWakeHandler(projects *services.ProjectService, routes RouteSyncer, serverDomain string, log *slog.Logger) *WakeHandler {
	return &WakeHandler{
		projects:     projects,
		routes:       routes,
		serverDomain: services.NormalizeProjectHost(serverDomain),
		log:          log,
	}
}

func (h *WakeHandler) Handle(c fiber.Ctx) error {
	host := services.NormalizeProjectHost(c.Hostname())
	if host == "" || host == h.serverDomain {
		return c.Next()
	}

	project, err := h.projects.RequestWakeByDomain(host)
	if err != nil {
		if errors.Is(err, services.ErrProjectNotFound) {
			return c.Next()
		}
		return c.Status(fiber.StatusInternalServerError).Type("html").SendString(statusPage("Unable to wake project", "Rivet could not prepare this project to wake."))
	}

	switch project.Status {
	case database.StatusSleeping, database.StatusWaking, database.StatusStarting, database.StatusDeploying:
		return c.Status(fiber.StatusAccepted).Type("html").SendString(wakePage(project.Name))
	case database.StatusRunning:
		h.syncRoutes(c.Context(), project.ID)
		return c.Status(fiber.StatusAccepted).Type("html").SendString(wakePage(project.Name))
	case database.StatusStopped:
		return c.Status(fiber.StatusServiceUnavailable).Type("html").SendString(statusPage("Project stopped", "This project is stopped and must be started from Rivet."))
	case database.StatusFailed:
		return c.Status(fiber.StatusServiceUnavailable).Type("html").SendString(statusPage("Project failed", "This project needs attention in Rivet before it can serve traffic."))
	default:
		return c.Next()
	}
}

func (h *WakeHandler) syncRoutes(ctx context.Context, projectID string) {
	if h.routes == nil {
		return
	}
	if err := h.routes.Sync(ctx); err != nil && h.log != nil {
		h.log.Error("failed to sync caddy routes from wake handler", "project_id", projectID, "err", err)
	}
}

func wakePage(projectName string) string {
	name := html.EscapeString(strings.TrimSpace(projectName))
	if name == "" {
		name = "Project"
	}

	return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta http-equiv="refresh" content="2">
  <title>Waking ` + name + `</title>
  <style>
    :root { color-scheme: light dark; font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    body { min-height: 100vh; margin: 0; display: grid; place-items: center; background: Canvas; color: CanvasText; }
    main { width: min(28rem, calc(100vw - 2rem)); text-align: center; }
    .loader { width: 2.5rem; height: 2.5rem; margin: 0 auto 1rem; border: 3px solid color-mix(in srgb, CanvasText 18%, transparent); border-top-color: CanvasText; border-radius: 999px; animation: spin 0.8s linear infinite; }
    h1 { margin: 0; font-size: 1.25rem; line-height: 1.4; }
    p { margin: 0.5rem 0 0; color: color-mix(in srgb, CanvasText 64%, transparent); font-size: 0.9375rem; line-height: 1.6; }
    @keyframes spin { to { transform: rotate(360deg); } }
  </style>
</head>
<body>
  <main>
    <div class="loader" aria-hidden="true"></div>
    <h1>Waking ` + name + `</h1>
    <p>This page will refresh automatically.</p>
  </main>
</body>
</html>`
}

func statusPage(title string, message string) string {
	return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>` + html.EscapeString(title) + `</title>
  <style>
    :root { color-scheme: light dark; font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    body { min-height: 100vh; margin: 0; display: grid; place-items: center; background: Canvas; color: CanvasText; }
    main { width: min(28rem, calc(100vw - 2rem)); text-align: center; }
    h1 { margin: 0; font-size: 1.25rem; line-height: 1.4; }
    p { margin: 0.5rem 0 0; color: color-mix(in srgb, CanvasText 64%, transparent); font-size: 0.9375rem; line-height: 1.6; }
  </style>
</head>
<body>
  <main>
    <h1>` + html.EscapeString(title) + `</h1>
    <p>` + html.EscapeString(message) + `</p>
  </main>
</body>
</html>`
}

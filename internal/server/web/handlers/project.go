package handlers

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/devmin8/rivet/internal/server/mapper"
	"github.com/devmin8/rivet/internal/server/services"
	"github.com/devmin8/rivet/internal/server/web/requestctx"
	"github.com/gofiber/fiber/v3"
)

type ProjectHandler struct {
	projectService *services.ProjectService
	routes         RouteSyncer
	log            *slog.Logger
}

type RouteSyncer interface {
	Sync(ctx context.Context) error
}

func NewProjectHandler(projectService *services.ProjectService, routes RouteSyncer, log *slog.Logger) *ProjectHandler {
	return &ProjectHandler{projectService: projectService, routes: routes, log: log}
}

func (h *ProjectHandler) ListProjects(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	projects, err := h.projectService.ListProjects(userID, strings.EqualFold(c.Query("include_deleted"), "true"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
			Error:   "internal_error",
			Message: "Unable to list projects.",
		})
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToListProjectsResponse(projects))
}

func (h *ProjectHandler) CreateProject(c fiber.Ctx) error {
	req := new(dtos.CreateProjectRequest)
	if err := c.Bind().Body(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dtos.ErrorResponse{
			Error:   "invalid_request",
			Message: "Request body is invalid.",
		})
	}

	userID, _ := requestctx.RequireUserID(c)

	project, err := h.projectService.CreateProject(services.CreateProjectRequest{
		Name:        req.Name,
		Domain:      req.Domain,
		Description: req.Description,
		Port:        req.Port,
		Platform:    req.Platform,
		CreatedByID: userID,
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
			Error:   "internal_error",
			Message: "Unable to create project.",
		})
	}

	h.syncRoutes(c.Context(), project.ID)

	return c.Status(fiber.StatusCreated).JSON(mapper.ToCreateProjectResponse(project))
}

func (h *ProjectHandler) GetProject(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	project, err := h.projectService.GetProject(c.Params("id"), userID)
	if err != nil {
		if errors.Is(err, services.ErrProjectNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dtos.ErrorResponse{
				Error:   "not_found",
				Message: "Project was not found.",
			})
		}
		if errors.Is(err, services.ErrProjectInactive) {
			return c.Status(fiber.StatusConflict).JSON(dtos.ErrorResponse{
				Error:   "project_inactive",
				Message: "Project is not active.",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
			Error:   "internal_error",
			Message: "Unable to get project.",
		})
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToCreateProjectResponse(project))
}

func (h *ProjectHandler) GetProjectStats(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	stats, err := h.projectService.ProjectRuntimeStats(c.Context(), userID, c.Query("ids"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
			Error:   "internal_error",
			Message: "Unable to get project stats.",
		})
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToProjectRuntimeStatsResponse(stats))
}

func (h *ProjectHandler) DeployProject(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	project, err := h.projectService.DeployProject(c.Context(), c.Params("id"), userID)
	if err != nil {
		return projectError(c, err, "Unable to deploy project.")
	}

	h.syncRoutes(c.Context(), project.ID)

	return c.Status(fiber.StatusOK).JSON(mapper.ToCreateProjectResponse(project))
}

func (h *ProjectHandler) StartProject(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	project, err := h.projectService.StartProject(c.Context(), c.Params("id"), userID)
	if err != nil {
		return projectError(c, err, "Unable to start project.")
	}

	h.syncRoutes(c.Context(), project.ID)

	return c.Status(fiber.StatusOK).JSON(mapper.ToCreateProjectResponse(project))
}

func (h *ProjectHandler) StopProject(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	project, err := h.projectService.StopProject(c.Context(), c.Params("id"), userID)
	if err != nil {
		return projectError(c, err, "Unable to stop project.")
	}

	h.syncRoutes(c.Context(), project.ID)

	return c.Status(fiber.StatusOK).JSON(mapper.ToCreateProjectResponse(project))
}

func (h *ProjectHandler) DeleteProject(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)
	projectID := c.Params("id")

	if err := h.projectService.DeleteProject(c.Context(), projectID, userID); err != nil {
		return projectError(c, err, "Unable to delete project.")
	}

	h.syncRoutes(c.Context(), projectID)

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *ProjectHandler) syncRoutes(ctx context.Context, projectID string) {
	if h.routes == nil {
		return
	}

	if err := h.routes.Sync(ctx); err != nil && h.log != nil {
		h.log.Error("failed to sync caddy routes", "project_id", projectID, "err", err)
	}
}

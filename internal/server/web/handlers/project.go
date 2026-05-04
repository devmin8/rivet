package handlers

import (
	"errors"
	"io"

	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/devmin8/rivet/internal/server/mapper"
	"github.com/devmin8/rivet/internal/server/services"
	"github.com/devmin8/rivet/internal/server/web/requestctx"
	"github.com/gofiber/fiber/v3"
)

type ProjectHandler struct {
	projectService *services.ProjectService
}

func NewProjectHandler(projectService *services.ProjectService) *ProjectHandler {
	return &ProjectHandler{projectService: projectService}
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
		Image:       req.Image,
		Platform:    req.Platform,
		CreatedByID: userID,
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
			Error:   "internal_error",
			Message: "Unable to create project.",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(mapper.ToCreateProjectResponse(project))
}

func (h *ProjectHandler) GetProject(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	project, err := h.projectService.GetProject(c.Params("id"), userID)
	if err != nil {
		return projectError(c, err, "Unable to get project.")
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToCreateProjectResponse(project))
}

func (h *ProjectHandler) ListProjects(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	projects, err := h.projectService.ListProjects(userID)
	if err != nil {
		return projectError(c, err, "Unable to list projects.")
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToListProjectsResponse(projects))
}

func (h *ProjectHandler) DeleteProject(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	if err := h.projectService.DeleteProject(c.Context(), c.Params("id"), userID); err != nil {
		return projectError(c, err, "Unable to delete project.")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *ProjectHandler) DeployProject(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	project, err := h.projectService.DeployProject(c.Context(), c.Params("id"), userID)
	if err != nil {
		return projectError(c, err, "Unable to deploy project.")
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToCreateProjectResponse(project))
}

func (h *ProjectHandler) StartProject(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	project, err := h.projectService.StartProject(c.Context(), c.Params("id"), userID)
	if err != nil {
		return projectError(c, err, "Unable to start project.")
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToCreateProjectResponse(project))
}

func (h *ProjectHandler) StopProject(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	project, err := h.projectService.StopProject(c.Context(), c.Params("id"), userID)
	if err != nil {
		return projectError(c, err, "Unable to stop project.")
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToCreateProjectResponse(project))
}

func (h *ProjectHandler) DeleteProjectContainer(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	project, err := h.projectService.DeleteProjectContainer(c.Context(), c.Params("id"), userID)
	if err != nil {
		return projectError(c, err, "Unable to delete project container.")
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToCreateProjectResponse(project))
}

func projectError(c fiber.Ctx, err error, message string) error {
	if errors.Is(err, services.ErrProjectNotFound) || errors.Is(err, services.ErrProjectInactive) {
		return c.Status(fiber.StatusNotFound).JSON(dtos.ErrorResponse{
			Error:   "not_found",
			Message: "Project was not found.",
		})
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return c.Status(fiber.StatusBadRequest).JSON(dtos.ErrorResponse{
			Error:   "invalid_request",
			Message: "Image tarball is invalid.",
		})
	}
	if errors.Is(err, services.ErrProjectImageMissing) {
		return c.Status(fiber.StatusConflict).JSON(dtos.ErrorResponse{
			Error:   "image_missing",
			Message: "Project image is missing.",
		})
	}
	if errors.Is(err, services.ErrProjectDeploymentInProgress) {
		return c.Status(fiber.StatusConflict).JSON(dtos.ErrorResponse{
			Error:   "deployment_in_progress",
			Message: "Project deployment is already in progress.",
		})
	}

	return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
		Error:   "internal_error",
		Message: message,
	})
}

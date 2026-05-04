package handlers

import (
	"errors"

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
		Description: req.Description,
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
		if errors.Is(err, services.ErrProjectNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dtos.ErrorResponse{
				Error:   "not_found",
				Message: "Project was not found.",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
			Error:   "internal_error",
			Message: "Unable to get project.",
		})
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToCreateProjectResponse(project))
}

func (h *ProjectHandler) ListProjects(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	projects, err := h.projectService.ListProjects(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
			Error:   "internal_error",
			Message: "Unable to list projects.",
		})
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToListProjectsResponse(projects))
}

func (h *ProjectHandler) DeleteProject(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	if err := h.projectService.DeleteProject(c.Params("id"), userID); err != nil {
		if errors.Is(err, services.ErrProjectNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dtos.ErrorResponse{
				Error:   "not_found",
				Message: "Project was not found.",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
			Error:   "internal_error",
			Message: "Unable to delete project.",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

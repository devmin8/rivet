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

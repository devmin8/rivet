package handlers

import (
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

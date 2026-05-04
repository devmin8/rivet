package handlers

import (
	"bytes"
	"errors"
	"io"
	"strings"

	"github.com/devmin8/rivet/internal/api"
	"github.com/devmin8/rivet/internal/api/dtos"
	rivetdocker "github.com/devmin8/rivet/internal/docker"
	"github.com/devmin8/rivet/internal/server/mapper"
	"github.com/devmin8/rivet/internal/server/services"
	"github.com/devmin8/rivet/internal/server/web/requestctx"
	"github.com/gofiber/fiber/v3"
)

type ImageHandler struct {
	projectService *services.ProjectService
	dockerClient   *rivetdocker.Client
}

func NewImageHandler(projectService *services.ProjectService, dockerClient *rivetdocker.Client) *ImageHandler {
	return &ImageHandler{
		projectService: projectService,
		dockerClient:   dockerClient,
	}
}

func (h *ImageHandler) UploadImage(c fiber.Ctx) error {
	projectID := strings.TrimSpace(c.Get(api.ImageProjectIDHeader))
	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dtos.ErrorResponse{
			Error:   "invalid_request",
			Message: "Project ID is required.",
		})
	}

	imageTag := strings.TrimSpace(c.Get(api.ImageTagHeader))
	if imageTag == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dtos.ErrorResponse{
			Error:   "invalid_request",
			Message: "Image tag is required.",
		})
	}

	body := c.Request().BodyStream()
	if body == nil {
		body = bytes.NewReader(c.Body())
	}
	if body == nil {
		return c.Status(fiber.StatusBadRequest).JSON(dtos.ErrorResponse{
			Error:   "invalid_request",
			Message: "Image tarball is required.",
		})
	}

	userID, _ := requestctx.RequireUserID(c)
	if _, err := h.projectService.GetProject(projectID, userID); err != nil {
		return projectError(c, err, "Unable to upload image.")
	}

	if err := h.dockerClient.LoadImage(c.Context(), body); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
			Error:   "docker_error",
			Message: "Unable to load image.",
		})
	}

	image := imageTag
	imageID, err := h.dockerClient.InspectImageID(c.Context(), imageTag)
	if err == nil && imageID != "" {
		image = imageID
	}

	project, err := h.projectService.UpdateProjectImage(projectID, userID, image)
	if err != nil {
		return projectError(c, err, "Unable to update project image.")
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToCreateProjectResponse(project))
}

func projectError(c fiber.Ctx, err error, message string) error {
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
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return c.Status(fiber.StatusBadRequest).JSON(dtos.ErrorResponse{
			Error:   "invalid_request",
			Message: "Image tarball is invalid.",
		})
	}

	return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
		Error:   "internal_error",
		Message: message,
	})
}

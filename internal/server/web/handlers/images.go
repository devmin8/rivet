package handlers

import (
	"bytes"
	"errors"
	"io"
	"strings"

	"github.com/devmin8/rivet/internal/api"
	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/devmin8/rivet/internal/docker"
	"github.com/devmin8/rivet/internal/server/services"
	"github.com/devmin8/rivet/internal/server/web/requestctx"
	"github.com/gofiber/fiber/v3"
)

type ImageHandler struct {
	appService   *services.AppService
	dockerClient *docker.Client
}

func NewImageHandler(appService *services.AppService, dockerClient *docker.Client) *ImageHandler {
	return &ImageHandler{
		appService:   appService,
		dockerClient: dockerClient,
	}
}

func (h *ImageHandler) UploadImage(c fiber.Ctx) error {
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
	if _, err := h.appService.GetApp(c.Params("appID"), userID); err != nil {
		return imageError(c, err, "Unable to upload image.")
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

	return c.Status(fiber.StatusOK).JSON(dtos.ImageUploadResponse{Image: image})
}

func imageError(c fiber.Ctx, err error, message string) error {
	if errors.Is(err, services.ErrAppNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(dtos.ErrorResponse{
			Error:   "not_found",
			Message: "App was not found.",
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

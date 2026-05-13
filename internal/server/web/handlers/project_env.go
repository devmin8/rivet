package handlers

import (
	"errors"

	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/devmin8/rivet/internal/server/mapper"
	"github.com/devmin8/rivet/internal/server/services"
	"github.com/devmin8/rivet/internal/server/web/requestctx"
	"github.com/gofiber/fiber/v3"
)

type ProjectEnvHandler struct {
	env *services.ProjectEnvService
}

func NewProjectEnvHandler(env *services.ProjectEnvService) *ProjectEnvHandler {
	return &ProjectEnvHandler{env: env}
}

func (h *ProjectEnvHandler) ListProjectEnv(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	items, err := h.env.ListProjectEnv(c.Params("id"), userID)
	if err != nil {
		return projectEnvError(c, err, "Unable to list project environment variables.")
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToListProjectEnvResponse(items))
}

func (h *ProjectEnvHandler) UpsertProjectEnv(c fiber.Ctx) error {
	req := new(dtos.UpsertProjectEnvRequest)
	if err := c.Bind().Body(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dtos.ErrorResponse{
			Error:   "invalid_request",
			Message: "Request body is invalid.",
		})
	}

	userID, _ := requestctx.RequireUserID(c)
	item, err := h.env.UpsertProjectEnv(c.Params("id"), userID, c.Params("key"), services.UpsertProjectEnvRequest{
		Kind:  string(req.Kind),
		Value: req.Value,
	})
	if err != nil {
		return projectEnvError(c, err, "Unable to save project environment variable.")
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToProjectEnvVarResponse(*item))
}

func (h *ProjectEnvHandler) DeleteProjectEnv(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	if err := h.env.DeleteProjectEnv(c.Params("id"), userID, c.Params("key")); err != nil {
		return projectEnvError(c, err, "Unable to delete project environment variable.")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func projectEnvError(c fiber.Ctx, err error, message string) error {
	if errors.Is(err, services.ErrInvalidEnvKey) {
		return c.Status(fiber.StatusBadRequest).JSON(dtos.ErrorResponse{
			Error:   "invalid_env_key",
			Message: "Environment variable keys must match ^[A-Za-z_][A-Za-z0-9_]*$.",
		})
	}
	if errors.Is(err, services.ErrReservedEnvKey) {
		return c.Status(fiber.StatusBadRequest).JSON(dtos.ErrorResponse{
			Error:   "reserved_env_key",
			Message: "Environment variable keys starting with RIVET_ are reserved.",
		})
	}
	if errors.Is(err, services.ErrInvalidEnvKind) {
		return c.Status(fiber.StatusBadRequest).JSON(dtos.ErrorResponse{
			Error:   "invalid_env_kind",
			Message: "Environment variable kind must be plain or secret.",
		})
	}
	if errors.Is(err, services.ErrMissingSecretKey) || errors.Is(err, services.ErrInvalidSecretKey) {
		return c.Status(fiber.StatusConflict).JSON(dtos.ErrorResponse{
			Error:   "secret_key_unavailable",
			Message: "RIVET_SECRET_KEY must be configured before saving or starting projects with secrets.",
		})
	}

	return projectError(c, err, message)
}

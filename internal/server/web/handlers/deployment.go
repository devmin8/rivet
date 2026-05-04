package handlers

import (
	"errors"

	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/devmin8/rivet/internal/server/mapper"
	"github.com/devmin8/rivet/internal/server/services"
	"github.com/devmin8/rivet/internal/server/web/requestctx"
	"github.com/gofiber/fiber/v3"
)

type DeploymentHandler struct {
	deploymentService *services.DeploymentService
}

func NewDeploymentHandler(deploymentService *services.DeploymentService) *DeploymentHandler {
	return &DeploymentHandler{deploymentService: deploymentService}
}

func (h *DeploymentHandler) CreateDeployment(c fiber.Ctx) error {
	req := new(dtos.CreateDeploymentRequest)
	if err := c.Bind().Body(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dtos.ErrorResponse{
			Error:   "invalid_request",
			Message: "Request body is invalid.",
		})
	}

	userID, _ := requestctx.RequireUserID(c)
	deployment, err := h.deploymentService.CreateDeployment(c.Context(), services.CreateDeploymentRequest{
		AppID:       c.Params("appID"),
		Image:       req.Image,
		CreatedByID: userID,
	})
	if err != nil {
		return deploymentError(c, err, "Unable to create deployment.")
	}

	return c.Status(fiber.StatusCreated).JSON(mapper.ToDeploymentResponse(deployment))
}

func (h *DeploymentHandler) ListDeployments(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	deployments, err := h.deploymentService.ListDeployments(c.Context(), c.Params("appID"), userID)
	if err != nil {
		return deploymentError(c, err, "Unable to list deployments.")
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToListDeploymentsResponse(deployments))
}

func (h *DeploymentHandler) StartAppContainer(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	deployment, err := h.deploymentService.StartAppContainer(c.Context(), c.Params("appID"), userID)
	if err != nil {
		return deploymentError(c, err, "Unable to start app.")
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToDeploymentResponse(deployment))
}

func (h *DeploymentHandler) StopAppContainer(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	deployment, err := h.deploymentService.StopAppContainer(c.Context(), c.Params("appID"), userID)
	if err != nil {
		return deploymentError(c, err, "Unable to stop app.")
	}
	if deployment == nil {
		return c.SendStatus(fiber.StatusNoContent)
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToDeploymentResponse(deployment))
}

func (h *DeploymentHandler) DeleteAppContainer(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	deployment, err := h.deploymentService.DeleteAppContainer(c.Context(), c.Params("appID"), userID)
	if err != nil {
		return deploymentError(c, err, "Unable to delete app container.")
	}
	if deployment == nil {
		return c.SendStatus(fiber.StatusNoContent)
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToDeploymentResponse(deployment))
}

func deploymentError(c fiber.Ctx, err error, message string) error {
	if errors.Is(err, services.ErrAppNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(dtos.ErrorResponse{
			Error:   "not_found",
			Message: "App was not found.",
		})
	}
	if errors.Is(err, services.ErrDeploymentNotFound) || errors.Is(err, services.ErrAppNoCurrentDeployment) {
		return c.Status(fiber.StatusNotFound).JSON(dtos.ErrorResponse{
			Error:   "not_found",
			Message: "Deployment was not found.",
		})
	}
	if errors.Is(err, services.ErrAppDeploymentInProgress) {
		return c.Status(fiber.StatusConflict).JSON(dtos.ErrorResponse{
			Error:   "conflict",
			Message: "App deployment is already in progress.",
		})
	}

	return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
		Error:   "internal_error",
		Message: message,
	})
}

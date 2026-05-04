package handlers

import (
	"errors"

	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/devmin8/rivet/internal/server/mapper"
	"github.com/devmin8/rivet/internal/server/services"
	"github.com/devmin8/rivet/internal/server/web/requestctx"
	"github.com/gofiber/fiber/v3"
)

type AppHandler struct {
	appService *services.AppService
}

func NewAppHandler(appService *services.AppService) *AppHandler {
	return &AppHandler{appService: appService}
}

func (h *AppHandler) CreateApp(c fiber.Ctx) error {
	req := new(dtos.CreateAppRequest)
	if err := c.Bind().Body(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dtos.ErrorResponse{
			Error:   "invalid_request",
			Message: "Request body is invalid.",
		})
	}

	userID, _ := requestctx.RequireUserID(c)
	app, err := h.appService.CreateApp(services.CreateAppRequest{
		ProjectID:   c.Params("projectID"),
		Name:        req.Name,
		Domain:      req.Domain,
		Description: req.Description,
		Port:        req.Port,
		Platform:    req.Platform,
		CreatedByID: userID,
	})
	if err != nil {
		return appError(c, err, "Unable to create app.")
	}

	return c.Status(fiber.StatusCreated).JSON(mapper.ToAppResponse(app))
}

func (h *AppHandler) GetApp(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	app, err := h.appService.GetApp(c.Params("appID"), userID)
	if err != nil {
		return appError(c, err, "Unable to get app.")
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToAppResponse(app))
}

func (h *AppHandler) ListApps(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	apps, err := h.appService.ListApps(c.Params("projectID"), userID)
	if err != nil {
		return appError(c, err, "Unable to list apps.")
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToListAppsResponse(apps))
}

func (h *AppHandler) DeleteApp(c fiber.Ctx) error {
	userID, _ := requestctx.RequireUserID(c)

	if err := h.appService.DeleteApp(c.Params("appID"), userID); err != nil {
		return appError(c, err, "Unable to delete app.")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func appError(c fiber.Ctx, err error, message string) error {
	if errors.Is(err, services.ErrProjectNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(dtos.ErrorResponse{
			Error:   "not_found",
			Message: "Project was not found.",
		})
	}
	if errors.Is(err, services.ErrAppNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(dtos.ErrorResponse{
			Error:   "not_found",
			Message: "App was not found.",
		})
	}

	return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
		Error:   "internal_error",
		Message: message,
	})
}

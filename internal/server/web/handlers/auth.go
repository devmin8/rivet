package handlers

import (
	"errors"

	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/devmin8/rivet/internal/server/mapper"
	"github.com/devmin8/rivet/internal/server/services"
	"github.com/gofiber/fiber/v3"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) RegisterUser(c fiber.Ctx) error {
	req := new(dtos.RegisterUserRequest)
	if err := c.Bind().Body(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dtos.ErrorResponse{
			Error:   "invalid_request",
			Message: "Request body is invalid.",
		})
	}

	user, err := h.authService.RegisterUser(req.Email, req.Username, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrUserAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(dtos.ErrorResponse{
				Error:   "registration_failed",
				Message: "Unable to register an account with the provided credentials.",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
			Error:   "internal_error",
			Message: "Unable to register account.",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(mapper.ToRegisterUserResponse(user))
}

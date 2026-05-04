package middlewares

import (
	"errors"

	"github.com/devmin8/rivet/internal/api"
	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/devmin8/rivet/internal/server/services"
	"github.com/devmin8/rivet/internal/server/web/requestctx"
	"github.com/gofiber/fiber/v3"
)

func RequireAuth(authService *services.AuthService) fiber.Handler {
	return func(c fiber.Ctx) error {
		token := c.Cookies(api.SessionCookieName)
		user, err := authService.UserFromSessionToken(token)
		if err != nil {
			if errors.Is(err, services.ErrInvalidSession) {
				return c.Status(fiber.StatusUnauthorized).JSON(dtos.ErrorResponse{
					Error:   "unauthorized",
					Message: "Authentication is required.",
				})
			}

			return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
				Error:   "internal_error",
				Message: "Unable to authenticate request.",
			})
		}

		requestctx.SetUserID(c, user.ID)
		return c.Next()
	}
}

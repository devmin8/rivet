package middlewares

import (
	"crypto/subtle"

	"github.com/devmin8/rivet/internal/api"
	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/gofiber/fiber/v3"
)

func RequireCSRF() fiber.Handler {
	return func(c fiber.Ctx) error {
		if isSafeMethod(c.Method()) {
			return c.Next()
		}

		cookieToken := c.Cookies(api.CSRFCookieName)
		headerToken := c.Get(api.CSRFHeaderName)
		// Compare in constant time so token mismatches do not leak timing hints.
		if cookieToken == "" || headerToken == "" || subtle.ConstantTimeCompare([]byte(cookieToken), []byte(headerToken)) != 1 {
			return c.Status(fiber.StatusForbidden).JSON(dtos.ErrorResponse{
				Error:   "csrf_failed",
				Message: "Request verification failed.",
			})
		}

		return c.Next()
	}
}

func isSafeMethod(method string) bool {
	return method == fiber.MethodGet ||
		method == fiber.MethodHead ||
		method == fiber.MethodOptions ||
		method == fiber.MethodTrace
}

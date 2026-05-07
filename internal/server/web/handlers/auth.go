package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"github.com/devmin8/rivet/internal/api"
	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/devmin8/rivet/internal/server/mapper"
	"github.com/devmin8/rivet/internal/server/services"
	"github.com/devmin8/rivet/internal/server/web/requestctx"
	"github.com/gofiber/fiber/v3"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) RegisterUser(c fiber.Ctx) error {
	if !isJSONRequest(c) {
		return c.Status(fiber.StatusUnsupportedMediaType).JSON(dtos.ErrorResponse{
			Error:   "unsupported_media_type",
			Message: "Request body must be JSON.",
		})
	}

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
		if errors.Is(err, services.ErrRegistrationClosed) {
			return c.Status(fiber.StatusForbidden).JSON(dtos.ErrorResponse{
				Error:   "registration_closed",
				Message: "Registration is closed because the admin user already exists.",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
			Error:   "internal_error",
			Message: "Unable to register account.",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(mapper.ToRegisterUserResponse(user))
}

func (h *AuthHandler) SignInUser(c fiber.Ctx) error {
	if !isJSONRequest(c) {
		return c.Status(fiber.StatusUnsupportedMediaType).JSON(dtos.ErrorResponse{
			Error:   "unsupported_media_type",
			Message: "Request body must be JSON.",
		})
	}

	req := new(dtos.SignInUserRequest)
	if err := c.Bind().Body(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dtos.ErrorResponse{
			Error:   "invalid_request",
			Message: "Request body is invalid.",
		})
	}

	result, err := h.authService.SignInUser(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(dtos.ErrorResponse{
				Error:   "signin_failed",
				Message: "Unable to sign in with the provided credentials.",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
			Error:   "internal_error",
			Message: "Unable to sign in.",
		})
	}

	c.Cookie(&fiber.Cookie{
		Name:        api.SessionCookieName,
		Value:       result.SessionToken,
		Path:        "/",
		Secure:      true,
		HTTPOnly:    true,
		SameSite:    fiber.CookieSameSiteStrictMode,
		SessionOnly: true,
	})

	csrfToken, err := newCSRFToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
			Error:   "internal_error",
			Message: "Unable to sign in.",
		})
	}

	setCSRFCookie(c, csrfToken)

	return c.Status(fiber.StatusOK).JSON(mapper.ToSignInUserResponse(result.User, csrfToken))
}

func (h *AuthHandler) CurrentUser(c fiber.Ctx) error {
	userID, err := requestctx.RequireUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dtos.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication is required.",
		})
	}

	csrfToken := c.Cookies(api.CSRFCookieName)
	if csrfToken == "" {
		csrfToken, err = newCSRFToken()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(dtos.ErrorResponse{
				Error:   "internal_error",
				Message: "Unable to verify session.",
			})
		}

		setCSRFCookie(c, csrfToken)
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToCurrentUserResponse(userID, csrfToken))
}

func isJSONRequest(c fiber.Ctx) bool {
	contentType := strings.ToLower(c.Get(fiber.HeaderContentType))
	return strings.HasPrefix(contentType, fiber.MIMEApplicationJSON)
}

func setCSRFCookie(c fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:        api.CSRFCookieName,
		Value:       token,
		Path:        "/",
		Secure:      true,
		HTTPOnly:    false,
		SameSite:    fiber.CookieSameSiteStrictMode,
		SessionOnly: true,
	})
}

func newCSRFToken() (string, error) {
	token := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, token); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(token), nil
}

package requestctx

import (
	"errors"

	"github.com/gofiber/fiber/v3"
)

var ErrMissingUserID = errors.New("missing user id in request context")

const currentUserIDKey = "current_user_id"

func SetUserID(c fiber.Ctx, userID string) {
	c.Locals(currentUserIDKey, userID)
}

func RequireUserID(c fiber.Ctx) (string, error) {
	userID, ok := c.Locals(currentUserIDKey).(string)
	if !ok || userID == "" {
		return "", ErrMissingUserID
	}

	return userID, nil
}

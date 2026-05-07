package mapper

import (
	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/devmin8/rivet/internal/server/database"
)

func ToRegisterUserResponse(user *database.User) dtos.RegisterUserResponse {
	return dtos.RegisterUserResponse{
		ID: user.ID,
	}
}

func ToSignInUserResponse(user *database.User, csrfToken string) dtos.SignInUserResponse {
	return dtos.SignInUserResponse{
		ID:        user.ID,
		CSRFToken: csrfToken,
	}
}

func ToCurrentUserResponse(userID string, csrfToken string) dtos.CurrentUserResponse {
	return dtos.CurrentUserResponse{
		ID:        userID,
		CSRFToken: csrfToken,
	}
}

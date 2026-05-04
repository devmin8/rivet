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

func ToSignInUserResponse(user *database.User) dtos.SignInUserResponse {
	return dtos.SignInUserResponse{
		ID: user.ID,
	}
}

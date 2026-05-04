package dtos

type RegisterUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=32,username"`
	// OWASP Authentication Cheat Sheet recommends at least 15 chars when MFA is not enabled,
	// no composition rules, and a max of at least 64 chars for passphrases:
	// https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html
	Password string `json:"password" validate:"required,min=15,max=128"`
	Email    string `json:"email" validate:"required,email,max=255"`
}

type RegisterUserResponse struct {
	ID string `json:"id"`
}

type SignInUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=32,username"`
	// Keep signin password validation aligned with registration.
	Password string `json:"password" validate:"required,min=15,max=128"`
}

type SignInUserResponse struct {
	ID string `json:"id"`
}

package services

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/devmin8/rivet/internal/server/database"
	"golang.org/x/crypto/argon2"
	"gorm.io/gorm"
)

var ErrUserAlreadyExists = errors.New("user already exists")

type AuthService struct {
	db  *gorm.DB
	log *slog.Logger
}

func NewAuthService(db *gorm.DB, log *slog.Logger) *AuthService {
	return &AuthService{db: db, log: log}
}

func (s *AuthService) RegisterUser(email string, username string, password string) (*database.User, error) {
	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &database.User{
		Username:     strings.ToLower(strings.TrimSpace(username)),
		Email:        strings.ToLower(strings.TrimSpace(email)),
		PasswordHash: passwordHash,
	}

	if err := s.db.Create(user).Error; err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	return user, nil
}

const (
	argon2Memory  = 19 * 1024
	argon2Time    = 2
	argon2Threads = 1
	argon2KeyLen  = 32
	saltLen       = 16
)

func hashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	// OWASP Password Storage Cheat Sheet recommends Argon2id with at least
	// m=19 MiB, t=2, p=1 for password storage:
	// https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
	key := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedKey := base64.RawStdEncoding.EncodeToString(key)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory, argon2Time, argon2Threads, encodedSalt, encodedKey), nil
}

func isUniqueConstraintError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	return strings.Contains(err.Error(), "UNIQUE constraint failed") ||
		strings.Contains(err.Error(), "duplicated key not allowed")
}

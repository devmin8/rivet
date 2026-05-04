package services

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/devmin8/rivet/internal/server/database"
	"golang.org/x/crypto/argon2"
	"gorm.io/gorm"
)

var ErrUserAlreadyExists = errors.New("user already exists")
var ErrInvalidCredentials = errors.New("invalid credentials")

type AuthService struct {
	db  *gorm.DB
	log *slog.Logger
}

type SignInResult struct {
	User         *database.User
	SessionToken string
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

func (s *AuthService) SignInUser(username string, password string) (*SignInResult, error) {
	var user database.User
	normalizedUsername := strings.ToLower(strings.TrimSpace(username))
	if err := s.db.Where("username = ?", normalizedUsername).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrInvalidCredentials
	}

	ok, err := verifyPassword(user.PasswordHash, password)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidCredentials
	}

	sessionToken, err := newSessionToken()
	if err != nil {
		return nil, err
	}

	sessionData, err := json.Marshal(sessionData{UserID: user.ID})
	if err != nil {
		return nil, err
	}

	session := &database.Session{
		ID:         sessionIDFromToken(sessionToken),
		Data:       string(sessionData),
		ExpiryDate: time.Now().UTC().Add(sessionTTL),
	}
	if err := s.db.Create(session).Error; err != nil {
		return nil, err
	}

	return &SignInResult{
		User:         &user,
		SessionToken: sessionToken,
	}, nil
}

const (
	argon2Memory  = 19 * 1024
	argon2Time    = 2
	argon2Threads = 1
	argon2KeyLen  = 32
	saltLen       = 16
	sessionTTL    = 8 * time.Hour
	sessionLen    = 32
)

type sessionData struct {
	UserID string `json:"user_id"`
}

// hashPassword stores passwords as self-describing Argon2id hashes:
// algorithm/version/params/salt/key all live in the returned string.
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

// newSessionToken returns the opaque secret sent to the browser cookie.
// The raw token is never stored server-side.
func newSessionToken() (string, error) {
	token := make([]byte, sessionLen)
	if _, err := io.ReadFull(rand.Reader, token); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(token), nil
}

// sessionIDFromToken derives the database lookup key from a cookie token.
// Storing only this digest prevents DB reads from exposing live sessions.
func sessionIDFromToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

// verifyPassword recomputes the Argon2id key using the stored hash metadata,
// then compares keys in constant time to avoid timing leaks.
func verifyPassword(encodedHash string, password string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false, nil
	}

	memory, iterations, threads, err := parseArgon2Params(parts[3])
	if err != nil {
		return false, nil
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, nil
	}

	expectedKey, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, nil
	}

	key := argon2.IDKey([]byte(password), salt, iterations, memory, threads, uint32(len(expectedKey)))
	return subtle.ConstantTimeCompare(key, expectedKey) == 1, nil
}

// parseArgon2Params extracts m/t/p values from the encoded hash so password
// verification uses the exact parameters chosen when the password was stored.
func parseArgon2Params(encodedParams string) (uint32, uint32, uint8, error) {
	var memory uint32
	var iterations uint32
	var threads uint8

	for _, param := range strings.Split(encodedParams, ",") {
		key, value, ok := strings.Cut(param, "=")
		if !ok {
			return 0, 0, 0, fmt.Errorf("invalid argon2 parameter: %s", param)
		}

		parsed, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return 0, 0, 0, err
		}

		switch key {
		case "m":
			memory = uint32(parsed)
		case "t":
			iterations = uint32(parsed)
		case "p":
			if parsed > 255 {
				return 0, 0, 0, fmt.Errorf("argon2 parallelism exceeds uint8: %d", parsed)
			}
			threads = uint8(parsed)
		default:
			return 0, 0, 0, fmt.Errorf("unknown argon2 parameter: %s", key)
		}
	}

	if memory == 0 || iterations == 0 || threads == 0 {
		return 0, 0, 0, errors.New("missing argon2 parameters")
	}

	return memory, iterations, threads, nil
}

func isUniqueConstraintError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	return strings.Contains(err.Error(), "UNIQUE constraint failed") ||
		strings.Contains(err.Error(), "duplicated key not allowed")
}

package services

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/devmin8/rivet/internal/server/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvalidEnvKey = errors.New("invalid environment variable key")
var ErrReservedEnvKey = errors.New("environment variable key uses a reserved prefix")
var ErrInvalidEnvKind = errors.New("invalid environment variable kind")
var ErrMissingSecretKey = errors.New("RIVET_SECRET_KEY is required for project secrets")
var ErrInvalidSecretKey = errors.New("RIVET_SECRET_KEY is invalid")

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type ProjectEnvService struct {
	db        *gorm.DB
	secretKey []byte
}

type UpsertProjectEnvRequest struct {
	Kind  string
	Value string
}

func NewProjectEnvService(db *gorm.DB, secretKey []byte) *ProjectEnvService {
	return &ProjectEnvService{db: db, secretKey: secretKey}
}

func (s *ProjectEnvService) ListProjectEnv(projectID string, userID string) ([]database.ProjectEnvVar, error) {
	if err := s.requireProject(projectID, userID); err != nil {
		return nil, err
	}

	var items []database.ProjectEnvVar
	err := s.db.
		Where("project_id = ?", projectID).
		Order("key ASC").
		Find(&items).
		Error
	return items, err
}

func (s *ProjectEnvService) UpsertProjectEnv(projectID string, userID string, key string, req UpsertProjectEnvRequest) (*database.ProjectEnvVar, error) {
	key = strings.TrimSpace(key)
	if err := validateEnvKey(key); err != nil {
		return nil, err
	}

	kind := database.ProjectEnvKind(strings.TrimSpace(req.Kind))
	if !kind.Valid() {
		return nil, ErrInvalidEnvKind
	}

	if err := s.requireProject(projectID, userID); err != nil {
		return nil, err
	}

	item := database.ProjectEnvVar{
		ProjectID:  projectID,
		Key:        key,
		Kind:       kind,
		KeyVersion: 1,
	}

	switch kind {
	case database.ProjectEnvKindPlain:
		item.Value = req.Value
	case database.ProjectEnvKindSecret:
		encrypted, err := s.encrypt(req.Value)
		if err != nil {
			return nil, err
		}
		item.EncryptedValue = encrypted
	}

	err := s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "project_id"}, {Name: "key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"kind":            item.Kind,
			"value":           item.Value,
			"encrypted_value": item.EncryptedValue,
			"key_version":     item.KeyVersion,
			"updated_at":      time.Now().UTC(),
		}),
	}).Create(&item).Error
	if err != nil {
		return nil, err
	}

	var saved database.ProjectEnvVar
	if err := s.db.Where("project_id = ? AND key = ?", projectID, key).First(&saved).Error; err != nil {
		return nil, err
	}

	return &saved, nil
}

func (s *ProjectEnvService) DeleteProjectEnv(projectID string, userID string, key string) error {
	key = strings.TrimSpace(key)
	if err := validateEnvKey(key); err != nil {
		return err
	}
	if err := s.requireProject(projectID, userID); err != nil {
		return err
	}

	return s.db.Where("project_id = ? AND key = ?", projectID, key).Delete(&database.ProjectEnvVar{}).Error
}

func (s *ProjectEnvService) RuntimeEnv(projectID string) ([]string, error) {
	var items []database.ProjectEnvVar
	if err := s.db.Where("project_id = ?", projectID).Find(&items).Error; err != nil {
		return nil, err
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Key < items[j].Key
	})

	env := make([]string, 0, len(items))
	for _, item := range items {
		value := item.Value
		if item.Kind == database.ProjectEnvKindSecret {
			decrypted, err := s.decrypt(item.EncryptedValue)
			if err != nil {
				return nil, fmt.Errorf("load project runtime env: %w", err)
			}
			value = decrypted
		}

		env = append(env, item.Key+"="+value)
	}

	return env, nil
}

func (s *ProjectEnvService) requireProject(projectID string, userID string) error {
	var project database.Project
	if err := s.db.Where("id = ? AND created_by_id = ?", projectID, userID).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProjectNotFound
		}
		return err
	}
	if !project.IsActive {
		return ErrProjectInactive
	}

	return nil
}

// validateEnvKey keeps project-provided keys safe to persist and later pass to
// Docker as KEY=value entries. Examples: PORT, DATABASE_URL, and _TOKEN are
// valid; 1PORT, DATABASE-URL, DATABASE URL, and RIVET_SECRET_KEY are rejected.
func validateEnvKey(key string) error {
	if !envKeyPattern.MatchString(key) {
		return ErrInvalidEnvKey
	}
	if strings.HasPrefix(key, "RIVET_") {
		return ErrReservedEnvKey
	}

	return nil
}

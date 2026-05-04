package services

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/devmin8/rivet/internal/server/database"
	"gorm.io/gorm"
)

var ErrAppNotFound = errors.New("app not found")

type AppService struct {
	db *gorm.DB
}

type CreateAppRequest struct {
	ProjectID   string
	Name        string
	Domain      string
	Description string
	Port        uint32
	Platform    string
	CreatedByID string
}

func NewAppService(db *gorm.DB) *AppService {
	return &AppService{db: db}
}

func (s *AppService) CreateApp(req CreateAppRequest) (*database.App, error) {
	if err := ensureProjectOwnedByUser(s.db, req.ProjectID, req.CreatedByID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	app := &database.App{
		ProjectID:       req.ProjectID,
		Name:            strings.TrimSpace(req.Name),
		Domain:          strings.TrimSpace(req.Domain),
		Description:     strings.TrimSpace(req.Description),
		Port:            strconv.FormatUint(uint64(req.Port), 10),
		Platform:        normalizePlatform(req.Platform),
		Status:          database.AppStatusStopped,
		DesiredStatus:   database.DesiredStatusStopped,
		StatusUpdatedAt: now,
		CreatedByID:     req.CreatedByID,
		UpdatedByID:     req.CreatedByID,
	}
	// TODO: persist app env vars when service config supports them.

	if err := s.db.Create(app).Error; err != nil {
		return nil, err
	}

	return app, nil
}

func (s *AppService) GetApp(id string, userID string) (*database.App, error) {
	return getAppForUser(s.db, id, userID)
}

func (s *AppService) ListApps(projectID string, userID string) ([]database.App, error) {
	if err := ensureProjectOwnedByUser(s.db, projectID, userID); err != nil {
		return nil, err
	}

	var apps []database.App
	if err := s.db.Where("project_id = ?", projectID).Order("created_at DESC").Find(&apps).Error; err != nil {
		return nil, err
	}

	return apps, nil
}

func (s *AppService) DeleteApp(id string, userID string) error {
	app, err := getAppForUser(s.db, id, userID)
	if err != nil {
		return err
	}

	if err := s.db.Delete(app).Error; err != nil {
		return err
	}

	// TODO: sync Caddy routes after app deletion.
	return nil
}

func ensureProjectOwnedByUser(db *gorm.DB, projectID string, userID string) error {
	var project database.Project
	if err := db.Where("id = ? AND created_by_id = ?", projectID, userID).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProjectNotFound
		}
		return err
	}

	return nil
}

func getAppForUser(db *gorm.DB, id string, userID string) (*database.App, error) {
	var app database.App
	err := db.Joins("JOIN projects ON projects.id = apps.project_id").
		Where("apps.id = ? AND projects.created_by_id = ?", id, userID).
		First(&app).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAppNotFound
		}
		return nil, err
	}

	return &app, nil
}

func normalizePlatform(platform string) database.Platform {
	value := database.Platform(strings.TrimSpace(platform))
	if value.Valid() {
		return value
	}

	return database.PlatformLinuxAMD64
}

package services

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/devmin8/rivet/internal/docker"
	"github.com/devmin8/rivet/internal/server/database"
	"gorm.io/gorm"
)

var ErrProjectNotFound = errors.New("project not found")
var ErrProjectInactive = errors.New("project is not active")
var ErrDeployInProgress = errors.New("deploy is already in progress")
var ErrNoTargetImage = errors.New("project has no target image")

type ProjectService struct {
	db     *gorm.DB
	docker *docker.Client
}

type CreateProjectRequest struct {
	Name        string
	Domain      string
	Description string
	Port        uint32
	Platform    string
	CreatedByID string
}

func NewProjectService(db *gorm.DB, docker *docker.Client) *ProjectService {
	return &ProjectService{db: db, docker: docker}
}

func (s *ProjectService) CreateProject(req CreateProjectRequest) (*database.Project, error) {
	project := &database.Project{
		Name:          strings.TrimSpace(req.Name),
		Domain:        strings.TrimSpace(req.Domain),
		Description:   strings.TrimSpace(req.Description),
		Port:          strconv.FormatUint(uint64(req.Port), 10),
		Platform:      normalizePlatform(req.Platform),
		Status:        database.StatusStopped,
		DesiredStatus: database.DesiredStatusStopped,
		CreatedByID:   req.CreatedByID,
		UpdatedByID:   req.CreatedByID,
	}

	if err := s.db.Create(project).Error; err != nil {
		return nil, err
	}

	// TODO: sync Caddy using project domain and deterministic container name.
	return project, nil
}

func (s *ProjectService) GetProject(id string, userID string) (*database.Project, error) {
	var project database.Project
	if err := s.db.Where("id = ? AND created_by_id = ?", id, userID).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	if !project.IsActive {
		return nil, ErrProjectInactive
	}

	return &project, nil
}

func (s *ProjectService) UpdateProjectTargetImage(id string, userID string, image string) (*database.Project, error) {
	res := s.db.Model(&database.Project{}).
		Where("id = ? AND created_by_id = ? AND is_active = ? AND status <> ?", id, userID, true, database.StatusDeploying).
		Update("target_image_ref", strings.TrimSpace(image))

	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, s.classifyProjectGuardFailure(id, userID, true)
	}

	return s.GetProject(id, userID)
}

func (s *ProjectService) DeployProject(ctx context.Context, id string, userID string) (*database.Project, error) {
	project, err := s.markDeploying(id, userID)
	if err != nil {
		return nil, err
	}

	if err := s.docker.RemoveContainer(ctx, project.ContainerName); err != nil {
		return nil, s.markDeployFailed(project.ID, project.TargetImageRef, err)
	}

	containerID, err := s.docker.StartContainer(ctx, project.ContainerName, project.ID, project.TargetImageRef)
	if err != nil {
		return nil, s.markDeployFailed(project.ID, project.TargetImageRef, err)
	}

	return s.markDeployRunning(project.ID, project.TargetImageRef, containerID)
}

func (s *ProjectService) ActiveDesiredRunningProjects() ([]database.Project, error) {
	var projects []database.Project
	err := s.db.Where("is_active = ? AND desired_status = ?", true, database.DesiredStatusRunning).Find(&projects).Error
	return projects, err
}

func (s *ProjectService) MarkReconciledRunning(id string, targetImage string, containerID string) error {
	return s.db.Model(&database.Project{}).
		Where("id = ? AND target_image_ref = ?", id, targetImage).
		Updates(map[string]any{
			"current_image_ref": targetImage,
			"status":            database.StatusRunning,
			"desired_status":    database.DesiredStatusRunning,
			"container_id":      containerID,
			"last_error":        "",
			"last_active_at":    time.Now().UTC(),
		}).Error
}

func (s *ProjectService) MarkReconcileFailed(id string, targetImage string, containerID string, err error) error {
	updates := map[string]any{
		"status":         database.StatusFailed,
		"desired_status": database.DesiredStatusRunning,
		"last_error":     err.Error(),
	}
	if containerID != "" {
		updates["container_id"] = containerID
	}

	return s.db.Model(&database.Project{}).
		Where("id = ? AND target_image_ref = ?", id, targetImage).
		Updates(updates).Error
}

func (s *ProjectService) markDeploying(id string, userID string) (*database.Project, error) {
	res := s.db.Model(&database.Project{}).
		Where("id = ? AND created_by_id = ? AND is_active = ? AND status <> ? AND target_image_ref <> ?", id, userID, true, database.StatusDeploying, "").
		Updates(map[string]any{
			"status":         database.StatusDeploying,
			"desired_status": database.DesiredStatusRunning,
			"last_error":     "",
		})

	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		// This branch turns the atomic guard miss into the public API error.
		return nil, s.classifyProjectGuardFailure(id, userID, false)
	}

	return s.GetProject(id, userID)
}

func (s *ProjectService) markDeployRunning(id string, targetImage string, containerID string) (*database.Project, error) {
	now := time.Now().UTC()
	if err := s.db.Model(&database.Project{}).
		Where("id = ? AND target_image_ref = ?", id, targetImage).
		Updates(map[string]any{
			"current_image_ref": targetImage,
			"status":            database.StatusRunning,
			"desired_status":    database.DesiredStatusRunning,
			"container_id":      containerID,
			"last_error":        "",
			"last_active_at":    now,
		}).Error; err != nil {
		return nil, err
	}

	var project database.Project
	if err := s.db.Where("id = ?", id).First(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (s *ProjectService) markDeployFailed(id string, targetImage string, deployErr error) error {
	if err := s.db.Model(&database.Project{}).
		Where("id = ? AND target_image_ref = ?", id, targetImage).
		Updates(map[string]any{
			"status":         database.StatusFailed,
			"desired_status": database.DesiredStatusRunning,
			"last_error":     deployErr.Error(),
		}).Error; err != nil {
		return errors.Join(deployErr, err)
	}

	return deployErr
}

func (s *ProjectService) classifyProjectGuardFailure(id string, userID string, allowNoTarget bool) error {
	var project database.Project
	if err := s.db.Where("id = ? AND created_by_id = ?", id, userID).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProjectNotFound
		}
		return err
	}
	if !project.IsActive {
		return ErrProjectInactive
	}
	if project.Status == database.StatusDeploying {
		return ErrDeployInProgress
	}
	if !allowNoTarget && strings.TrimSpace(project.TargetImageRef) == "" {
		return ErrNoTargetImage
	}

	return ErrProjectNotFound
}

func normalizePlatform(platform string) database.Platform {
	value := database.Platform(strings.TrimSpace(platform))
	if value.Valid() {
		return value
	}

	return database.PlatformLinuxAMD64
}

package services

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	rivetdocker "github.com/devmin8/rivet/internal/docker"
	"github.com/devmin8/rivet/internal/server/database"
	"gorm.io/gorm"
)

var ErrProjectNotFound = errors.New("project not found")
var ErrProjectInactive = errors.New("project is not active")
var ErrProjectImageMissing = errors.New("project image is missing")
var ErrProjectDeploymentInProgress = errors.New("project deployment is already in progress")

type ProjectService struct {
	db     *gorm.DB
	docker *rivetdocker.Client
}

type CreateProjectRequest struct {
	Name        string
	Domain      string
	Description string
	Port        uint32
	Image       string
	Platform    string
	CreatedByID string
}

func NewProjectService(db *gorm.DB, dockerClient ...*rivetdocker.Client) *ProjectService {
	var docker *rivetdocker.Client
	if len(dockerClient) > 0 {
		docker = dockerClient[0]
	}

	return &ProjectService{db: db, docker: docker}
}

func (s *ProjectService) CreateProject(req CreateProjectRequest) (*database.Project, error) {
	now := time.Now().UTC()
	project := &database.Project{
		Name:            strings.TrimSpace(req.Name),
		Domain:          strings.TrimSpace(req.Domain),
		Description:     strings.TrimSpace(req.Description),
		Port:            strconv.FormatUint(uint64(req.Port), 10),
		Image:           strings.TrimSpace(req.Image),
		Platform:        normalizePlatform(req.Platform),
		Status:          database.StatusStopped,
		DesiredStatus:   database.DesiredStatusStopped,
		StatusUpdatedAt: now,
		IsActive:        true,
		CreatedByID:     req.CreatedByID,
		UpdatedByID:     req.CreatedByID,
	}

	if err := s.db.Create(project).Error; err != nil {
		return nil, err
	}

	// TODO: sync Caddy routes after project config is created.
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

func (s *ProjectService) ListProjects(userID string) ([]database.Project, error) {
	var projects []database.Project
	if err := s.db.Where("created_by_id = ? AND is_active = ?", userID, true).Order("created_at DESC").Find(&projects).Error; err != nil {
		return nil, err
	}

	return projects, nil
}

func (s *ProjectService) DeleteProject(ctx context.Context, id string, userID string) error {
	project, err := s.GetProject(id, userID)
	if err != nil {
		return err
	}

	if s.docker != nil && project.ContainerID != "" {
		if err := s.docker.RemoveContainer(ctx, project.ContainerID); err != nil {
			return err
		}
	}

	now := time.Now().UTC()
	if err := s.db.Model(project).Updates(map[string]any{
		"is_active":         false,
		"status":            database.StatusStopped,
		"desired_status":    database.DesiredStatusStopped,
		"status_updated_at": statusUpdatedAtExpr(database.StatusStopped, now),
		"container_id":      "",
		"error":             "",
		"updated_by_id":     userID,
	}).Error; err != nil {
		return err
	}

	// TODO: sync Caddy routes after project deletion.
	return nil
}

func (s *ProjectService) UpdateProjectImage(id string, userID string, image string) (*database.Project, error) {
	project, err := s.GetProject(id, userID)
	if err != nil {
		return nil, err
	}
	if project.Status == database.StatusCreating {
		return nil, ErrProjectDeploymentInProgress
	}

	updates := map[string]any{
		"image":         strings.TrimSpace(image),
		"error":         "",
		"updated_by_id": userID,
	}

	if project.Status != database.StatusRunning {
		updates["status"] = database.StatusStopped
		updates["desired_status"] = database.DesiredStatusStopped
		updates["status_updated_at"] = time.Now().UTC()
	}

	if err := s.db.Model(project).Updates(updates).Error; err != nil {
		return nil, err
	}

	return s.GetProject(id, userID)
}

func (s *ProjectService) DeployProject(ctx context.Context, id string, userID string) (*database.Project, error) {
	return s.runProject(ctx, id, userID)
}

func (s *ProjectService) StartProject(ctx context.Context, id string, userID string) (*database.Project, error) {
	return s.runProject(ctx, id, userID)
}

func (s *ProjectService) runProject(ctx context.Context, id string, userID string) (*database.Project, error) {
	project, err := s.GetProject(id, userID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(project.Image) == "" {
		return nil, ErrProjectImageMissing
	}

	claimed, err := s.claimProjectRun(project.ID, userID)
	if err != nil {
		return nil, err
	}
	if !claimed {
		return nil, ErrProjectDeploymentInProgress
	}

	if err := s.removeProjectContainer(ctx, project.ContainerID); err != nil {
		if markErr := s.markProjectFailedIfDesiredRunning(project.ID, userID, err); markErr != nil {
			return nil, errors.Join(err, markErr)
		}
		return nil, err
	}
	if err := s.clearProjectContainerID(project.ID, project.ContainerID); err != nil {
		return nil, err
	}
	project.ContainerID = ""

	containerID, err := s.startProjectContainer(ctx, project)
	if err != nil {
		if markErr := s.markProjectFailedIfDesiredRunning(project.ID, userID, err); markErr != nil {
			return nil, errors.Join(err, markErr)
		}
		return nil, err
	}

	running, err := s.markProjectRunningIfDesiredRunning(project.ID, userID, containerID)
	if err != nil {
		if removeErr := s.removeProjectContainer(ctx, containerID); removeErr != nil {
			return nil, errors.Join(err, removeErr)
		}
		return nil, err
	}
	if !running {
		if err := s.removeProjectContainer(ctx, containerID); err != nil {
			return nil, err
		}
	}

	// TODO: sync Caddy routes so project.Domain points to this container and project.Port.
	return s.GetProject(id, userID)
}

func (s *ProjectService) StopProject(ctx context.Context, id string, userID string) (*database.Project, error) {
	project, err := s.GetProject(id, userID)
	if err != nil {
		return nil, err
	}

	if err := s.markProjectDesiredStopped(project.ID, userID); err != nil {
		return nil, err
	}

	if s.docker != nil && project.ContainerID != "" {
		if err := s.docker.StopContainer(ctx, project.ContainerID); err != nil {
			return nil, err
		}
	}
	if err := s.markProjectStoppedIfDesiredStopped(project.ID, userID, false); err != nil {
		return nil, err
	}

	// TODO: sync Caddy routes after project is intentionally stopped.
	return s.GetProject(id, userID)
}

func (s *ProjectService) DeleteProjectContainer(ctx context.Context, id string, userID string) (*database.Project, error) {
	project, err := s.GetProject(id, userID)
	if err != nil {
		return nil, err
	}

	if err := s.markProjectDesiredStopped(project.ID, userID); err != nil {
		return nil, err
	}

	if err := s.removeProjectContainer(ctx, project.ContainerID); err != nil {
		return nil, err
	}
	if err := s.markProjectStoppedIfDesiredStopped(project.ID, userID, true); err != nil {
		return nil, err
	}

	// TODO: sync Caddy routes after project container is removed.
	return s.GetProject(id, userID)
}

func (s *ProjectService) claimProjectRun(id string, userID string) (bool, error) {
	now := time.Now().UTC()
	res := s.db.Model(&database.Project{}).
		Where("id = ? AND created_by_id = ? AND is_active = ? AND status <> ?", id, userID, true, database.StatusCreating).
		Updates(map[string]any{
			"status":            database.StatusCreating,
			"desired_status":    database.DesiredStatusRunning,
			"status_updated_at": now,
			"error":             "",
			"updated_by_id":     userID,
		})
	if res.Error != nil {
		return false, res.Error
	}

	return res.RowsAffected == 1, nil
}

func (s *ProjectService) markProjectRunningIfDesiredRunning(id string, userID string, containerID string) (bool, error) {
	now := time.Now().UTC()
	res := s.db.Model(&database.Project{}).
		Where("id = ? AND created_by_id = ? AND is_active = ? AND desired_status = ? AND status = ?", id, userID, true, database.DesiredStatusRunning, database.StatusCreating).
		Updates(map[string]any{
			"container_id":      containerID,
			"status":            database.StatusRunning,
			"status_updated_at": now,
			"error":             "",
			"last_active_at":    now,
			"updated_by_id":     userID,
		})
	if res.Error != nil {
		return false, res.Error
	}

	return res.RowsAffected == 1, nil
}

func (s *ProjectService) markProjectDesiredStopped(id string, userID string) error {
	return s.db.Model(&database.Project{}).
		Where("id = ? AND created_by_id = ? AND is_active = ?", id, userID, true).
		Updates(map[string]any{
			"desired_status": database.DesiredStatusStopped,
			"error":          "",
			"updated_by_id":  userID,
		}).Error
}

func (s *ProjectService) markProjectStoppedIfDesiredStopped(id string, userID string, clearContainer bool) error {
	now := time.Now().UTC()
	updates := map[string]any{
		"status":            database.StatusStopped,
		"desired_status":    database.DesiredStatusStopped,
		"status_updated_at": statusUpdatedAtExpr(database.StatusStopped, now),
		"error":             "",
		"updated_by_id":     userID,
	}
	if clearContainer {
		updates["container_id"] = ""
	}

	return s.db.Model(&database.Project{}).
		Where("id = ? AND created_by_id = ? AND is_active = ? AND desired_status = ?", id, userID, true, database.DesiredStatusStopped).
		Updates(updates).Error
}

func (s *ProjectService) markProjectFailedIfDesiredRunning(id string, userID string, cause error) error {
	now := time.Now().UTC()
	return s.db.Model(&database.Project{}).
		Where("id = ? AND created_by_id = ? AND is_active = ? AND desired_status = ?", id, userID, true, database.DesiredStatusRunning).
		Updates(map[string]any{
			"status":            database.StatusFailed,
			"status_updated_at": now,
			"error":             cause.Error(),
			"updated_by_id":     userID,
		}).Error
}

func (s *ProjectService) startProjectContainer(ctx context.Context, project *database.Project) (string, error) {
	if s.docker == nil {
		return "", errors.New("docker client is not configured")
	}

	// TODO: load project env vars and pass them into StartContainerOptions.Env.
	return s.docker.StartContainer(ctx, rivetdocker.StartContainerOptions{
		Image:     project.Image,
		Platform:  string(project.Platform),
		Env:       nil,
		ProjectID: project.ID,
	})
}

func (s *ProjectService) removeProjectContainer(ctx context.Context, containerID string) error {
	if s.docker == nil || strings.TrimSpace(containerID) == "" {
		return nil
	}

	return s.docker.RemoveContainer(ctx, containerID)
}

func (s *ProjectService) clearProjectContainerID(id string, containerID string) error {
	return s.db.Model(&database.Project{}).
		Where("id = ? AND container_id = ?", id, containerID).
		Update("container_id", "").Error
}

func statusUpdatedAtExpr(target database.Status, now time.Time) any {
	return gorm.Expr("CASE WHEN status <> ? THEN ? ELSE status_updated_at END", target, now)
}

func normalizePlatform(platform string) database.Platform {
	value := database.Platform(strings.TrimSpace(platform))
	if value.Valid() {
		return value
	}

	return database.PlatformLinuxAMD64
}

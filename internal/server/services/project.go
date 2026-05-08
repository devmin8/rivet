package services

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
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
	log    *slog.Logger

	// lifecycleMu serializes project lifecycle actions while the MVP state model has no starting/stopping states.
	lifecycleMu sync.Mutex
	// statsMu prevents concurrent dashboard requests from stampeding Docker when the stats cache expires.
	statsMu sync.Mutex
	// statsCache stores recent runtime stats and raw CPU samples used to calculate the next CPU percent.
	statsCache map[string]projectRuntimeStatsCacheEntry
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
	return NewProjectServiceWithLogger(db, docker, nil)
}

func NewProjectServiceWithLogger(db *gorm.DB, docker *docker.Client, log *slog.Logger) *ProjectService {
	return &ProjectService{
		db:         db,
		docker:     docker,
		log:        log,
		statsCache: make(map[string]projectRuntimeStatsCacheEntry),
	}
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

	return project, nil
}

func (s *ProjectService) ListProjects(userID string, includeDeleted bool) ([]database.Project, error) {
	query := s.db.Where("created_by_id = ?", userID)
	if !includeDeleted {
		query = query.Where("is_active = ?", true)
	}

	var projects []database.Project
	err := query.Order("created_at DESC").Find(&projects).Error
	return projects, err
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

func (s *ProjectService) StartProject(ctx context.Context, id string, userID string) (*database.Project, error) {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	project, err := s.GetProject(id, userID)
	if err != nil {
		return nil, err
	}
	if project.Status == database.StatusDeploying {
		return nil, ErrDeployInProgress
	}

	image := strings.TrimSpace(project.TargetImageRef)
	if image == "" {
		image = strings.TrimSpace(project.CurrentImageRef)
	}
	if image == "" {
		return nil, ErrNoTargetImage
	}

	if project.Status == database.StatusRunning && project.DesiredStatus == database.DesiredStatusRunning {
		return project, nil
	}

	s.clearRuntimeStatsCache(project.ID)
	if err := s.docker.RemoveContainer(ctx, project.ContainerName); err != nil {
		return nil, err
	}

	containerID, err := s.docker.StartContainer(ctx, project.ContainerName, project.ID, image)
	if err != nil {
		return nil, err
	}

	return s.markStarted(project.ID, userID, image, containerID)
}

func (s *ProjectService) StopProject(ctx context.Context, id string, userID string) (*database.Project, error) {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	project, err := s.GetProject(id, userID)
	if err != nil {
		return nil, err
	}
	if project.Status == database.StatusDeploying {
		return nil, ErrDeployInProgress
	}

	if project.Status == database.StatusStopped && project.DesiredStatus == database.DesiredStatusStopped && project.ContainerID == "" {
		return project, nil
	}

	s.clearRuntimeStatsCache(project.ID)
	if err := s.docker.RemoveContainer(ctx, project.ContainerName); err != nil {
		return nil, err
	}

	return s.markStopped(project.ID, userID)
}

func (s *ProjectService) DeleteProject(ctx context.Context, id string, userID string) error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	project, err := s.getProjectForOwner(id, userID, true)
	if err != nil {
		return err
	}
	if project.Status == database.StatusDeploying {
		return ErrDeployInProgress
	}

	s.clearRuntimeStatsCache(project.ID)
	if err := s.docker.RemoveContainer(ctx, project.ContainerName); err != nil {
		return err
	}

	res := s.db.Model(&database.Project{}).
		Where("id = ? AND created_by_id = ? AND status <> ?", id, userID, database.StatusDeploying).
		Updates(map[string]any{
			"is_active":      false,
			"status":         database.StatusStopped,
			"desired_status": database.DesiredStatusStopped,
			"container_id":   "",
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return s.classifyProjectGuardFailure(id, userID, true)
	}

	return nil
}

func (s *ProjectService) DeployProject(ctx context.Context, id string, userID string) (*database.Project, error) {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	project, err := s.markDeploying(id, userID)
	if err != nil {
		return nil, err
	}
	s.clearRuntimeStatsCache(project.ID)

	if err := s.docker.RemoveContainer(ctx, project.ContainerName); err != nil {
		return nil, s.markDeployFailed(project.ID, project.TargetImageRef, err)
	}

	containerID, err := s.docker.StartContainer(ctx, project.ContainerName, project.ID, project.TargetImageRef)
	if err != nil {
		return nil, s.markDeployFailed(project.ID, project.TargetImageRef, err)
	}

	project, err = s.markDeployRunning(project.ID, project.TargetImageRef, containerID)
	if err != nil {
		return nil, err
	}

	return project, nil
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

func (s *ProjectService) markStarted(id string, userID string, image string, containerID string) (*database.Project, error) {
	now := time.Now().UTC()
	res := s.db.Model(&database.Project{}).
		Where("id = ? AND created_by_id = ? AND is_active = ? AND status <> ?", id, userID, true, database.StatusDeploying).
		Updates(map[string]any{
			"current_image_ref": image,
			"status":            database.StatusRunning,
			"desired_status":    database.DesiredStatusRunning,
			"container_id":      containerID,
			"last_error":        "",
			"last_active_at":    now,
		})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, s.classifyProjectGuardFailure(id, userID, true)
	}

	return s.GetProject(id, userID)
}

func (s *ProjectService) markStopped(id string, userID string) (*database.Project, error) {
	res := s.db.Model(&database.Project{}).
		Where("id = ? AND created_by_id = ? AND is_active = ? AND status <> ?", id, userID, true, database.StatusDeploying).
		Updates(map[string]any{
			"status":         database.StatusStopped,
			"desired_status": database.DesiredStatusStopped,
			"container_id":   "",
		})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, s.classifyProjectGuardFailure(id, userID, true)
	}

	return s.GetProject(id, userID)
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
	project, err := s.getProjectForOwner(id, userID, false)
	if err != nil {
		return err
	}
	if project.Status == database.StatusDeploying {
		return ErrDeployInProgress
	}
	if !allowNoTarget && strings.TrimSpace(project.TargetImageRef) == "" {
		return ErrNoTargetImage
	}

	return ErrProjectNotFound
}

func (s *ProjectService) getProjectForOwner(id string, userID string, includeInactive bool) (*database.Project, error) {
	var project database.Project
	if err := s.db.Where("id = ? AND created_by_id = ?", id, userID).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	if !includeInactive && !project.IsActive {
		return nil, ErrProjectInactive
	}

	return &project, nil
}

func normalizePlatform(platform string) database.Platform {
	value := database.Platform(strings.TrimSpace(platform))
	if value.Valid() {
		return value
	}

	return database.PlatformLinuxAMD64
}

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
var ErrInvalidAutoSleepAfter = errors.New("auto sleep duration must be at least 60000 ms")
var ErrProjectStateChanged = errors.New("project runtime state changed")

const defaultAutoSleepAfterMS int64 = 60_000 // 1 minute
const minAutoSleepAfterMS int64 = 60_000     // 1 minute

type ProjectService struct {
	db     *gorm.DB
	docker *docker.Client
	log    *slog.Logger
	env    *ProjectEnvService

	// lifecycleMu serializes project lifecycle actions while start/stop/deploy run synchronously.
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

type UpdateProjectRuntimeSettingsRequest struct {
	AutoSleepAfterMS *int64
}

func NewProjectService(db *gorm.DB, docker *docker.Client, secretKey []byte) *ProjectService {
	return NewProjectServiceWithLogger(db, docker, secretKey, nil)
}

func NewProjectServiceWithLogger(db *gorm.DB, docker *docker.Client, secretKey []byte, log *slog.Logger) *ProjectService {
	return &ProjectService{
		db:         db,
		docker:     docker,
		log:        log,
		env:        NewProjectEnvService(db, secretKey),
		statsCache: make(map[string]projectRuntimeStatsCacheEntry),
	}
}

func (s *ProjectService) CreateProject(req CreateProjectRequest) (*database.Project, error) {
	// todo: get it from the cli, so cli needs to be updated
	autoSleepAfterMS := defaultAutoSleepAfterMS
	project := &database.Project{
		Name:             strings.TrimSpace(req.Name),
		Domain:           strings.TrimSpace(req.Domain),
		Description:      strings.TrimSpace(req.Description),
		Port:             strconv.FormatUint(uint64(req.Port), 10),
		Platform:         normalizePlatform(req.Platform),
		Status:           database.StatusStopped,
		AutoSleepAfterMS: &autoSleepAfterMS,
		CreatedByID:      req.CreatedByID,
		UpdatedByID:      req.CreatedByID,
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

// GetActiveProjectByDomain is used for public project traffic. At the Caddy/wake
// boundary, the request only has a Host header, not an authenticated project ID.
func (s *ProjectService) GetActiveProjectByDomain(domain string) (*database.Project, error) {
	var project database.Project
	if err := s.db.Where("domain = ? AND is_active = ?", strings.TrimSpace(domain), true).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	return &project, nil
}

// RequestWakeByDomain turns the public project host into a wake request without
// coupling the unauthenticated wake path to dashboard/API project IDs.
func (s *ProjectService) RequestWakeByDomain(domain string) (*database.Project, error) {
	res := s.db.Model(&database.Project{}).
		Where("domain = ? AND is_active = ? AND status = ?", strings.TrimSpace(domain), true, database.StatusSleeping).
		Updates(map[string]any{
			"status":     database.StatusWaking,
			"last_error": "",
		})
	if res.Error != nil {
		return nil, res.Error
	}

	return s.GetActiveProjectByDomain(domain)
}

func (s *ProjectService) UpdateProjectRuntimeSettings(id string, userID string, req UpdateProjectRuntimeSettingsRequest) (*database.Project, error) {
	if req.AutoSleepAfterMS != nil && *req.AutoSleepAfterMS < minAutoSleepAfterMS {
		return nil, ErrInvalidAutoSleepAfter
	}

	res := s.db.Model(&database.Project{}).
		Where("id = ? AND created_by_id = ? AND is_active = ?", id, userID, true).
		Update("auto_sleep_after_ms", req.AutoSleepAfterMS)

	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, s.classifyProjectGuardFailure(id, userID, true)
	}

	return s.GetProject(id, userID)
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

	if project.Status == database.StatusRunning {
		return project, nil
	}

	s.clearRuntimeStatsCache(project.ID)
	if err := s.docker.RemoveContainer(ctx, project.ContainerName); err != nil {
		return nil, s.markRuntimeFailed(project.ID, "", err)
	}

	env, err := s.env.RuntimeEnv(project.ID)
	if err != nil {
		return nil, s.markRuntimeFailed(project.ID, "", err)
	}

	containerID, err := s.docker.StartContainer(ctx, project.ContainerName, project.ID, image, env)
	if err != nil {
		return nil, s.markRuntimeFailed(project.ID, "", err)
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

	if project.Status == database.StatusStopped && project.ContainerID == "" {
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
			"is_active":    false,
			"status":       database.StatusStopped,
			"container_id": "",
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
		return nil, s.markRuntimeFailed(project.ID, "", err)
	}

	env, err := s.env.RuntimeEnv(project.ID)
	if err != nil {
		return nil, s.markRuntimeFailed(project.ID, "", err)
	}

	containerID, err := s.docker.StartContainer(ctx, project.ContainerName, project.ID, project.TargetImageRef, env)
	if err != nil {
		return nil, s.markRuntimeFailed(project.ID, "", err)
	}

	project, err = s.markDeployRunning(project.ID, project.TargetImageRef, containerID)
	if err != nil {
		return nil, err
	}

	return project, nil
}

func (s *ProjectService) ActiveRuntimeProjects() ([]database.Project, error) {
	var projects []database.Project
	err := s.db.
		Where("is_active = ? AND status IN ?", true, []database.Status{
			database.StatusStarting,
			database.StatusRunning,
		}).
		Find(&projects).Error
	return projects, err
}

func (s *ProjectService) WakingProjects() ([]database.Project, error) {
	var projects []database.Project
	err := s.db.
		Where("is_active = ? AND status = ?", true, database.StatusWaking).
		Find(&projects).Error
	return projects, err
}

func (s *ProjectService) RunningAutoSleepProjects() ([]database.Project, error) {
	var projects []database.Project
	err := s.db.
		Where("is_active = ? AND status = ? AND auto_sleep_after_ms IS NOT NULL", true, database.StatusRunning).
		Find(&projects).Error
	return projects, err
}

func (s *ProjectService) SleepingProjectsWithContainer() ([]database.Project, error) {
	var projects []database.Project
	err := s.db.
		Where("is_active = ? AND status = ? AND container_id <> ?", true, database.StatusSleeping, "").
		Find(&projects).Error
	return projects, err
}

func (s *ProjectService) MarkSleeping(id string) error {
	res := s.db.Model(&database.Project{}).
		Where("id = ? AND status = ?", id, database.StatusRunning).
		Updates(map[string]any{
			"status":     database.StatusSleeping,
			"last_error": "",
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrProjectStateChanged
	}

	return nil
}

func (s *ProjectService) MarkSleepAborted(id string, sleepErr error) error {
	return s.db.Model(&database.Project{}).
		Where("id = ? AND status = ?", id, database.StatusSleeping).
		Updates(map[string]any{
			"status":     database.StatusRunning,
			"last_error": sleepErr.Error(),
		}).Error
}

func (s *ProjectService) MarkSlept(id string) error {
	return s.db.Model(&database.Project{}).
		Where("id = ? AND status = ?", id, database.StatusSleeping).
		Update("container_id", "").Error
}

func (s *ProjectService) MarkWakeRunning(id string, targetImage string, containerID string) error {
	res := s.db.Model(&database.Project{}).
		Where("id = ? AND status = ?", id, database.StatusWaking).
		Updates(map[string]any{
			"current_image_ref": targetImage,
			"status":            database.StatusRunning,
			"container_id":      containerID,
			"last_error":        "",
			"last_active_at":    time.Now().UTC(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrProjectStateChanged
	}

	return nil
}

func (s *ProjectService) MarkWakeFailed(id string, err error) error {
	return s.db.Model(&database.Project{}).
		Where("id = ? AND status = ?", id, database.StatusWaking).
		Updates(map[string]any{
			"status":       database.StatusFailed,
			"last_error":   err.Error(),
			"container_id": "",
		}).Error
}

func (s *ProjectService) MarkReconciledRunning(id string, targetImage string, containerID string) error {
	return s.db.Model(&database.Project{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"current_image_ref": targetImage,
			"status":            database.StatusRunning,
			"container_id":      containerID,
			"last_error":        "",
		}).Error
}

func (s *ProjectService) MarkReconcileFailed(id string, containerID string, err error) error {
	updates := map[string]any{
		"status":       database.StatusFailed,
		"last_error":   err.Error(),
		"container_id": containerID,
	}

	return s.db.Model(&database.Project{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (s *ProjectService) markDeploying(id string, userID string) (*database.Project, error) {
	res := s.db.Model(&database.Project{}).
		Where("id = ? AND created_by_id = ? AND is_active = ? AND status <> ? AND target_image_ref <> ?", id, userID, true, database.StatusDeploying, "").
		Updates(map[string]any{
			"status":     database.StatusDeploying,
			"last_error": "",
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
			"status":       database.StatusStopped,
			"container_id": "",
		})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, s.classifyProjectGuardFailure(id, userID, true)
	}

	return s.GetProject(id, userID)
}

func (s *ProjectService) markRuntimeFailed(id string, containerID string, runtimeErr error) error {
	if err := s.MarkReconcileFailed(id, containerID, runtimeErr); err != nil {
		return errors.Join(runtimeErr, err)
	}

	return runtimeErr
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

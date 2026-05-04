package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/devmin8/rivet/internal/docker"
	"github.com/devmin8/rivet/internal/server/database"
	"gorm.io/gorm"
)

var ErrDeploymentNotFound = errors.New("deployment not found")
var ErrAppNoCurrentDeployment = errors.New("app has no current deployment")
var ErrAppDeploymentInProgress = errors.New("app deployment is already in progress")

type DeploymentService struct {
	db     *gorm.DB
	docker *docker.Client
}

type CreateDeploymentRequest struct {
	AppID       string
	Image       string
	CreatedByID string
}

func NewDeploymentService(db *gorm.DB, docker *docker.Client) *DeploymentService {
	return &DeploymentService{db: db, docker: docker}
}

func (s *DeploymentService) CreateDeployment(ctx context.Context, req CreateDeploymentRequest) (*database.Deployment, error) {
	image := strings.TrimSpace(req.Image)

	db := s.db.WithContext(ctx)
	app, err := getAppForUser(db, req.AppID, req.CreatedByID)
	if err != nil {
		return nil, err
	}
	if app.Status == database.AppStatusCreating {
		return nil, ErrAppDeploymentInProgress
	}

	now := time.Now().UTC()
	deployment := &database.Deployment{
		AppID:           app.ID,
		Image:           image,
		Status:          database.DeploymentStatusPending,
		StatusUpdatedAt: now,
		CreatedByID:     req.CreatedByID,
		UpdatedByID:     req.CreatedByID,
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(deployment).Error; err != nil {
			return err
		}

		return markAppCreating(tx, app.ID, deployment.ID, req.CreatedByID, now)
	}); err != nil {
		return nil, err
	}

	containerID, err := s.docker.StartContainer(ctx, docker.StartContainerOptions{
		Image:        deployment.Image,
		Platform:     string(app.Platform),
		Env:          nil, // TODO: pass app env vars into container start.
		AppID:        app.ID,
		DeploymentID: deployment.ID,
	})
	if err != nil {
		if markErr := s.markDeploymentFailed(db, deployment.ID, app.ID, req.CreatedByID, err); markErr != nil {
			return nil, errors.Join(err, markErr)
		}
		return nil, err
	}

	now = time.Now().UTC()
	if err := db.Transaction(func(tx *gorm.DB) error {
		return markDeploymentRunning(tx, app.ID, deployment.ID, containerID, req.CreatedByID, now)
	}); err != nil {
		return nil, err
	}

	// TODO: sync Caddy routes after deployment starts running.
	if err := db.First(deployment, "id = ?", deployment.ID).Error; err != nil {
		return nil, err
	}

	return deployment, nil
}

func (s *DeploymentService) ListDeployments(ctx context.Context, appID string, userID string) ([]database.Deployment, error) {
	db := s.db.WithContext(ctx)
	app, err := getAppForUser(db, appID, userID)
	if err != nil {
		return nil, err
	}

	var deployments []database.Deployment
	if err := db.Where("app_id = ?", app.ID).Order("created_at DESC").Find(&deployments).Error; err != nil {
		return nil, err
	}

	return deployments, nil
}

func (s *DeploymentService) StartAppContainer(ctx context.Context, appID string, userID string) (*database.Deployment, error) {
	db := s.db.WithContext(ctx)
	app, deployment, err := s.currentDeploymentForUser(db, appID, userID)
	if err != nil {
		return nil, err
	}
	if app.Status == database.AppStatusCreating {
		return nil, ErrAppDeploymentInProgress
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return markAppCreating(tx, app.ID, deployment.ID, userID, time.Now().UTC())
	}); err != nil {
		return nil, err
	}

	if deployment.ContainerID != "" {
		if err := s.docker.StopContainer(ctx, deployment.ContainerID); err != nil {
			return nil, err
		}
		if err := markDeploymentStopped(db, deployment.ID, userID, time.Now().UTC()); err != nil {
			return nil, err
		}
	}

	containerID, err := s.docker.StartContainer(ctx, docker.StartContainerOptions{
		Image:        deployment.Image,
		Platform:     string(app.Platform),
		Env:          nil, // TODO: pass app env vars into container start.
		AppID:        app.ID,
		DeploymentID: deployment.ID,
	})
	if err != nil {
		if markErr := s.markDeploymentFailed(db, deployment.ID, app.ID, userID, err); markErr != nil {
			return nil, errors.Join(err, markErr)
		}
		return nil, err
	}

	now := time.Now().UTC()
	if err := db.Transaction(func(tx *gorm.DB) error {
		return markDeploymentRunning(tx, app.ID, deployment.ID, containerID, userID, now)
	}); err != nil {
		return nil, err
	}

	// TODO: sync Caddy routes after app container starts.
	return s.getDeployment(db, deployment.ID)
}

func (s *DeploymentService) StopAppContainer(ctx context.Context, appID string, userID string) (*database.Deployment, error) {
	db := s.db.WithContext(ctx)
	app, deployment, err := s.currentDeploymentForUser(db, appID, userID)
	if err != nil && !errors.Is(err, ErrAppNoCurrentDeployment) {
		return nil, err
	}

	if deployment != nil && deployment.ContainerID != "" {
		if err := s.docker.StopContainer(ctx, deployment.ContainerID); err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	if err := db.Transaction(func(tx *gorm.DB) error {
		if deployment != nil {
			if err := markDeploymentStopped(tx, deployment.ID, userID, now); err != nil {
				return err
			}
		}

		return tx.Model(&database.App{}).Where("id = ?", app.ID).Updates(map[string]any{
			"status":            database.AppStatusStopped,
			"desired_status":    database.DesiredStatusStopped,
			"status_updated_at": statusUpdatedAtExpr(database.AppStatusStopped, now),
			"updated_by_id":     userID,
		}).Error
	}); err != nil {
		return nil, err
	}

	// TODO: sync Caddy routes after app container stops.
	if deployment == nil {
		return nil, nil
	}

	return s.getDeployment(db, deployment.ID)
}

func (s *DeploymentService) DeleteAppContainer(ctx context.Context, appID string, userID string) (*database.Deployment, error) {
	db := s.db.WithContext(ctx)
	app, deployment, err := s.currentDeploymentForUser(db, appID, userID)
	if err != nil && !errors.Is(err, ErrAppNoCurrentDeployment) {
		return nil, err
	}

	if deployment != nil && deployment.ContainerID != "" {
		if err := s.docker.RemoveContainer(ctx, deployment.ContainerID); err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	if err := db.Transaction(func(tx *gorm.DB) error {
		if deployment != nil {
			if err := tx.Model(&database.Deployment{}).Where("id = ?", deployment.ID).Updates(map[string]any{
				"container_id":      "",
				"status":            database.DeploymentStatusStopped,
				"status_updated_at": statusUpdatedAtExpr(database.DeploymentStatusStopped, now),
				"updated_by_id":     userID,
			}).Error; err != nil {
				return err
			}
		}

		return tx.Model(&database.App{}).Where("id = ?", app.ID).Updates(map[string]any{
			"status":            database.AppStatusStopped,
			"desired_status":    database.DesiredStatusStopped,
			"status_updated_at": statusUpdatedAtExpr(database.AppStatusStopped, now),
			"updated_by_id":     userID,
		}).Error
	}); err != nil {
		return nil, err
	}

	// TODO: sync Caddy routes after app container is deleted.
	if deployment == nil {
		return nil, nil
	}

	return s.getDeployment(db, deployment.ID)
}

func (s *DeploymentService) currentDeploymentForUser(db *gorm.DB, appID string, userID string) (*database.App, *database.Deployment, error) {
	app, err := getAppForUser(db, appID, userID)
	if err != nil {
		return nil, nil, err
	}
	if app.CurrentDeploymentID == nil || *app.CurrentDeploymentID == "" {
		return app, nil, ErrAppNoCurrentDeployment
	}

	deployment, err := s.getAppDeployment(db, app.ID, *app.CurrentDeploymentID)
	if err != nil {
		return app, nil, err
	}

	return app, deployment, nil
}

func (s *DeploymentService) getDeployment(db *gorm.DB, id string) (*database.Deployment, error) {
	var deployment database.Deployment
	if err := db.First(&deployment, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeploymentNotFound
		}
		return nil, err
	}

	return &deployment, nil
}

func (s *DeploymentService) getAppDeployment(db *gorm.DB, appID string, deploymentID string) (*database.Deployment, error) {
	var deployment database.Deployment
	if err := db.First(&deployment, "id = ? AND app_id = ?", deploymentID, appID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeploymentNotFound
		}
		return nil, err
	}

	return &deployment, nil
}

func (s *DeploymentService) markDeploymentFailed(db *gorm.DB, deploymentID string, appID string, userID string, cause error) error {
	now := time.Now().UTC()
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&database.Deployment{}).Where("id = ?", deploymentID).Updates(map[string]any{
			"status":            database.DeploymentStatusFailed,
			"status_updated_at": statusUpdatedAtExpr(database.DeploymentStatusFailed, now),
			"error":             cause.Error(),
			"updated_by_id":     userID,
		}).Error; err != nil {
			return err
		}

		return tx.Model(&database.App{}).Where("id = ? AND desired_status = ?", appID, database.DesiredStatusRunning).Updates(map[string]any{
			"status":            database.AppStatusFailed,
			"status_updated_at": statusUpdatedAtExpr(database.AppStatusFailed, now),
			"updated_by_id":     userID,
		}).Error
	})
}

func markDeploymentRunning(tx *gorm.DB, appID string, deploymentID string, containerID string, userID string, now time.Time) error {
	if err := tx.Model(&database.Deployment{}).Where("id = ?", deploymentID).Updates(map[string]any{
		"container_id":      containerID,
		"status":            database.DeploymentStatusRunning,
		"status_updated_at": statusUpdatedAtExpr(database.DeploymentStatusRunning, now),
		"error":             "",
		"updated_by_id":     userID,
	}).Error; err != nil {
		return err
	}

	return tx.Model(&database.App{}).Where("id = ?", appID).Updates(map[string]any{
		"status":            database.AppStatusRunning,
		"desired_status":    database.DesiredStatusRunning,
		"status_updated_at": statusUpdatedAtExpr(database.AppStatusRunning, now),
		"last_active_at":    now,
		"updated_by_id":     userID,
	}).Error
}

func markAppCreating(tx *gorm.DB, appID string, deploymentID string, userID string, now time.Time) error {
	res := tx.Model(&database.App{}).
		Where("id = ? AND status <> ?", appID, database.AppStatusCreating).
		Updates(map[string]any{
			"desired_status":        database.DesiredStatusRunning,
			"status":                database.AppStatusCreating,
			"status_updated_at":     statusUpdatedAtExpr(database.AppStatusCreating, now),
			"current_deployment_id": deploymentID,
			"updated_by_id":         userID,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAppDeploymentInProgress
	}

	return nil
}

func markDeploymentStopped(tx *gorm.DB, deploymentID string, userID string, now time.Time) error {
	return tx.Model(&database.Deployment{}).Where("id = ?", deploymentID).Updates(map[string]any{
		"status":            database.DeploymentStatusStopped,
		"status_updated_at": statusUpdatedAtExpr(database.DeploymentStatusStopped, now),
		"updated_by_id":     userID,
	}).Error
}

func statusUpdatedAtExpr(target any, now time.Time) any {
	return gorm.Expr("CASE WHEN status <> ? THEN ? ELSE status_updated_at END", target, now)
}

package services

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/devmin8/rivet/internal/docker"
	"github.com/devmin8/rivet/internal/server/database"
	"gorm.io/gorm"
)

const DefaultReconcileInterval = time.Minute
const DefaultStuckTimeout = 10 * time.Minute
const reconcileBatchSize = 100

type Reconciler struct {
	db           *gorm.DB
	docker       *docker.Client
	log          *slog.Logger
	interval     time.Duration
	stuckTimeout time.Duration
}

func NewReconciler(db *gorm.DB, docker *docker.Client, log *slog.Logger) *Reconciler {
	return &Reconciler{
		db:           db,
		docker:       docker,
		log:          log,
		interval:     DefaultReconcileInterval,
		stuckTimeout: DefaultStuckTimeout,
	}
}

func (r *Reconciler) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.Reconcile(ctx); err != nil {
				r.log.Error("reconcile failed", "err", err)
			}
		}
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	cutoff := time.Now().UTC().Add(-r.stuckTimeout)

	if err := r.failStuckPendingDeployments(ctx, cutoff); err != nil {
		return err
	}
	if err := r.failStuckCreatingApps(ctx, cutoff); err != nil {
		return err
	}
	if err := r.failImpossibleRunningApps(ctx); err != nil {
		return err
	}

	return r.reconcileLabeledContainers(ctx)
}

func (r *Reconciler) failStuckPendingDeployments(ctx context.Context, cutoff time.Time) error {
	var deployments []database.Deployment
	if err := r.db.WithContext(ctx).
		Select("id", "app_id").
		Where("status = ? AND status_updated_at < ?", database.DeploymentStatusPending, cutoff).
		Order("status_updated_at ASC").
		Limit(reconcileBatchSize).
		Find(&deployments).Error; err != nil {
		return err
	}

	now := time.Now().UTC()
	for i := range deployments {
		deployment := deployments[i]
		if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			res := tx.Model(&database.Deployment{}).
				Where("id = ? AND status = ? AND status_updated_at < ?", deployment.ID, database.DeploymentStatusPending, cutoff).
				Updates(map[string]any{
					"status":            database.DeploymentStatusFailed,
					"status_updated_at": statusUpdatedAtExpr(database.DeploymentStatusFailed, now),
					"error":             "deployment pending timeout",
				})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return nil
			}

			return tx.Model(&database.App{}).
				Where(
					"id = ? AND current_deployment_id = ? AND desired_status = ? AND status = ?",
					deployment.AppID,
					deployment.ID,
					database.DesiredStatusRunning,
					database.AppStatusCreating,
				).
				Updates(map[string]any{
					"status":            database.AppStatusFailed,
					"status_updated_at": statusUpdatedAtExpr(database.AppStatusFailed, now),
				}).Error
		}); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) failStuckCreatingApps(ctx context.Context, cutoff time.Time) error {
	return r.db.WithContext(ctx).Model(&database.App{}).
		Where("desired_status = ? AND status = ? AND status_updated_at < ?", database.DesiredStatusRunning, database.AppStatusCreating, cutoff).
		Updates(map[string]any{
			"status":            database.AppStatusFailed,
			"status_updated_at": statusUpdatedAtExpr(database.AppStatusFailed, time.Now().UTC()),
		}).Error
}

func (r *Reconciler) failImpossibleRunningApps(ctx context.Context) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&database.App{}).
		Where(
			`desired_status = ? AND status = ? AND (
				current_deployment_id IS NULL OR
				current_deployment_id = ? OR
				NOT EXISTS (
					SELECT 1
					FROM deployments
					WHERE deployments.id = apps.current_deployment_id
						AND deployments.app_id = apps.id
						AND deployments.status = ?
						AND deployments.container_id <> ?
				)
			)`,
			database.DesiredStatusRunning,
			database.AppStatusRunning,
			"",
			database.DeploymentStatusRunning,
			"",
		).
		Updates(map[string]any{
			"status":            database.AppStatusFailed,
			"status_updated_at": statusUpdatedAtExpr(database.AppStatusFailed, now),
		}).Error
}

func (r *Reconciler) reconcileLabeledContainers(ctx context.Context) error {
	containers, err := r.docker.ListRivetContainers(ctx)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for i := range containers {
		container := containers[i]
		if container.AppID == "" || container.DeploymentID == "" {
			continue
		}

		if err := r.reconcileLabeledContainer(ctx, container, now); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) reconcileLabeledContainer(ctx context.Context, container docker.Container, now time.Time) error {
	var app database.App
	if err := r.db.WithContext(ctx).First(&app, "id = ?", container.AppID).Error; err != nil {
		if errorsIsRecordNotFound(err) {
			return nil
		}
		return err
	}

	var deployment database.Deployment
	if err := r.db.WithContext(ctx).First(&deployment, "id = ? AND app_id = ?", container.DeploymentID, container.AppID).Error; err != nil {
		if errorsIsRecordNotFound(err) {
			return nil
		}
		return err
	}

	isCurrent := app.CurrentDeploymentID != nil && *app.CurrentDeploymentID == deployment.ID
	if container.Running && isCurrent && app.DesiredStatus == database.DesiredStatusStopped {
		if err := r.docker.StopContainer(ctx, container.ID); err != nil {
			return err
		}
		return r.markStoppedDeploymentAndApp(ctx, app.ID, deployment.ID, deployment.UpdatedByID, now)
	}

	if container.Running && isCurrent && app.DesiredStatus == database.DesiredStatusRunning {
		return r.markRecoveredDeploymentRunning(ctx, app.ID, deployment.ID, container.ID, now)
	}

	if !container.Running && deployment.Status == database.DeploymentStatusRunning {
		return r.failRunningDeployment(ctx, app.ID, deployment.ID, now)
	}

	if container.Running && !isCurrent {
		if err := r.docker.StopContainer(ctx, container.ID); err != nil {
			return err
		}
		return markDeploymentStopped(r.db.WithContext(ctx), deployment.ID, deployment.UpdatedByID, now)
	}

	return nil
}

func (r *Reconciler) markStoppedDeploymentAndApp(ctx context.Context, appID string, deploymentID string, userID string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := markDeploymentStopped(tx, deploymentID, userID, now); err != nil {
			return err
		}

		return tx.Model(&database.App{}).
			Where("id = ? AND current_deployment_id = ?", appID, deploymentID).
			Updates(map[string]any{
				"status":            database.AppStatusStopped,
				"status_updated_at": statusUpdatedAtExpr(database.AppStatusStopped, now),
			}).Error
	})
}

func (r *Reconciler) markRecoveredDeploymentRunning(ctx context.Context, appID string, deploymentID string, containerID string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&database.Deployment{}).
			Where("id = ?", deploymentID).
			Updates(map[string]any{
				"container_id":      containerID,
				"status":            database.DeploymentStatusRunning,
				"status_updated_at": statusUpdatedAtExpr(database.DeploymentStatusRunning, now),
				"error":             "",
			}).Error; err != nil {
			return err
		}

		return tx.Model(&database.App{}).
			Where("id = ? AND current_deployment_id = ? AND desired_status = ?", appID, deploymentID, database.DesiredStatusRunning).
			Updates(map[string]any{
				"status":            database.AppStatusRunning,
				"status_updated_at": statusUpdatedAtExpr(database.AppStatusRunning, now),
				"last_active_at":    now,
			}).Error
	})
}

func (r *Reconciler) failRunningDeployment(ctx context.Context, appID string, deploymentID string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&database.Deployment{}).
			Where("id = ? AND status = ?", deploymentID, database.DeploymentStatusRunning).
			Updates(map[string]any{
				"status":            database.DeploymentStatusFailed,
				"status_updated_at": statusUpdatedAtExpr(database.DeploymentStatusFailed, now),
				"error":             "container is not running",
			}).Error; err != nil {
			return err
		}

		return tx.Model(&database.App{}).
			Where("id = ? AND current_deployment_id = ? AND desired_status = ? AND status = ?", appID, deploymentID, database.DesiredStatusRunning, database.AppStatusRunning).
			Updates(map[string]any{
				"status":            database.AppStatusFailed,
				"status_updated_at": statusUpdatedAtExpr(database.AppStatusFailed, now),
			}).Error
	})
}

func errorsIsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/devmin8/rivet/internal/docker"
	"github.com/devmin8/rivet/internal/server/database"
	"gorm.io/gorm"
)

const reconcileInterval = 30 * time.Second

type Reconciler struct {
	projects *ProjectService
	docker   *docker.Client
	log      *slog.Logger
}

func NewReconciler(db *gorm.DB, dockerClient *docker.Client, log *slog.Logger) *Reconciler {
	return &Reconciler{
		projects: NewProjectService(db, dockerClient),
		docker:   dockerClient,
		log:      log,
	}
}

func (r *Reconciler) Start(ctx context.Context) {
	go r.run(ctx)
}

func (r *Reconciler) run(ctx context.Context) {
	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.reconcile(ctx)
		}
	}
}

func (r *Reconciler) reconcile(ctx context.Context) {
	projects, err := r.projects.ActiveDesiredRunningProjects()
	if err != nil {
		r.log.Error("failed to list projects for reconciliation", "err", err)
		return
	}

	for _, project := range projects {
		if err := r.reconcileProject(ctx, project); err != nil {
			r.log.Error("failed to reconcile project", "project_id", project.ID, "err", err)
		}
	}
}

func (r *Reconciler) reconcileProject(ctx context.Context, project database.Project) error {
	if project.TargetImageRef == "" {
		return r.projects.MarkReconcileFailed(project.ID, project.TargetImageRef, "", ErrNoTargetImage)
	}

	info, err := r.docker.InspectContainer(ctx, project.ContainerName)
	if errors.Is(err, docker.ErrContainerNotFound) {
		return r.startTargetOrRollback(ctx, project)
	}
	if err != nil {
		return r.projects.MarkReconcileFailed(project.ID, project.TargetImageRef, "", err)
	}

	if info.Running && info.Image == project.TargetImageRef {
		return r.projects.MarkReconciledRunning(project.ID, project.TargetImageRef, info.ID)
	}

	if info.Running && project.Status == database.StatusFailed && info.Image == project.CurrentImageRef {
		// A rollback is serving the last good image while the failed target stays desired.
		return nil
	}

	if err := r.docker.RemoveContainer(ctx, project.ContainerName); err != nil {
		return r.projects.MarkReconcileFailed(project.ID, project.TargetImageRef, info.ID, err)
	}

	return r.startTargetOrRollback(ctx, project)
}

func (r *Reconciler) startTargetOrRollback(ctx context.Context, project database.Project) error {
	containerID, err := r.docker.StartContainer(ctx, project.ContainerName, project.ID, project.TargetImageRef)
	if err == nil {
		return r.projects.MarkReconciledRunning(project.ID, project.TargetImageRef, containerID)
	}

	targetErr := fmt.Errorf("start target image %s: %w", project.TargetImageRef, err)
	if project.CurrentImageRef == "" {
		return r.projects.MarkReconcileFailed(project.ID, project.TargetImageRef, "", targetErr)
	}

	if err := r.docker.RemoveContainer(ctx, project.ContainerName); err != nil {
		// A failed start can leave a named container behind, which blocks rollback.
		return r.projects.MarkReconcileFailed(project.ID, project.TargetImageRef, "", errors.Join(targetErr, err))
	}

	rollbackID, rollbackErr := r.docker.StartContainer(ctx, project.ContainerName, project.ID, project.CurrentImageRef)
	if rollbackErr != nil {
		return r.projects.MarkReconcileFailed(project.ID, project.TargetImageRef, "", errors.Join(targetErr, rollbackErr))
	}

	return r.projects.MarkReconcileFailed(project.ID, project.TargetImageRef, rollbackID, targetErr)
}

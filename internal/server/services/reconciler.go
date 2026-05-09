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
	projects, err := r.projects.ActiveRuntimeProjects()
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
	expectedImage := expectedRuntimeImage(project)
	if expectedImage == "" {
		return r.projects.MarkReconcileFailed(project.ID, "", ErrNoTargetImage)
	}

	info, err := r.docker.InspectContainer(ctx, project.ContainerName)
	if errors.Is(err, docker.ErrContainerNotFound) {
		return r.projects.MarkReconcileFailed(project.ID, "", docker.ErrContainerNotFound)
	}
	if err != nil {
		return r.projects.MarkReconcileFailed(project.ID, "", err)
	}

	if info.Running && info.Image == expectedImage {
		return r.projects.MarkReconciledRunning(project.ID, expectedImage, info.ID)
	}

	if !info.Running {
		return r.projects.MarkReconcileFailed(project.ID, info.ID, docker.ErrContainerNotRunning)
	}

	// todo: at this point some image is running but the target image.
	// we need to handle this case, else user wont know easily which image is un necessary.
	err = errors.New("container image does not match project runtime state")
	return r.projects.MarkReconcileFailed(project.ID, info.ID, err)
}

func expectedRuntimeImage(project database.Project) string {
	switch project.Status {
	case database.StatusDeploying:
		return project.TargetImageRef
	default:
		if project.CurrentImageRef != "" {
			return project.CurrentImageRef
		}
		return project.TargetImageRef
	}
}

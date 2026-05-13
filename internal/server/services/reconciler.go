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
const sleepWakeInterval = 5 * time.Second

type Reconciler struct {
	projects *ProjectService
	docker   *docker.Client
	routes   routeSyncer
	log      *slog.Logger
}

type routeSyncer interface {
	Sync(ctx context.Context) error
}

func NewReconciler(db *gorm.DB, dockerClient *docker.Client, routes routeSyncer, secretKey []byte, log *slog.Logger) *Reconciler {
	return &Reconciler{
		projects: NewProjectService(db, dockerClient, secretKey),
		docker:   dockerClient,
		routes:   routes,
		log:      log,
	}
}

func (r *Reconciler) Start(ctx context.Context) {
	go r.run(ctx)
}

func (r *Reconciler) run(ctx context.Context) {
	ticker := time.NewTicker(sleepWakeInterval)
	defer ticker.Stop()

	lastReconcile := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.wakeProjects(ctx)
			r.sleepIdleProjects(ctx)
			r.cleanupSleepingContainers(ctx)

			if time.Since(lastReconcile) >= reconcileInterval {
				r.reconcile(ctx)
				lastReconcile = time.Now()
			}
		}
	}
}

func (r *Reconciler) wakeProjects(ctx context.Context) {
	projects, err := r.projects.WakingProjects()
	if err != nil {
		r.log.Error("failed to list waking projects", "err", err)
		return
	}

	for _, project := range projects {
		if err := r.wakeProject(ctx, project); err != nil {
			r.log.Error("failed to wake project", "project_id", project.ID, "err", err)
		}
	}
}

func (r *Reconciler) wakeProject(ctx context.Context, project database.Project) error {
	image := expectedRuntimeImage(project)
	if image == "" {
		if err := r.projects.MarkWakeFailed(project.ID, ErrNoTargetImage); err != nil {
			return err
		}
		return r.syncRoutes(ctx)
	}

	r.projects.clearRuntimeStatsCache(project.ID)
	// Remove any stale container with the project name before recreating it; missing containers are ignored by DockerClient.
	if err := r.docker.RemoveContainer(ctx, project.ContainerName); err != nil {
		if markErr := r.projects.MarkWakeFailed(project.ID, err); markErr != nil {
			return errors.Join(err, markErr)
		}
		return r.syncRoutes(ctx)
	}

	env, err := r.projects.env.RuntimeEnv(project.ID)
	if err != nil {
		if markErr := r.projects.MarkWakeFailed(project.ID, err); markErr != nil {
			return errors.Join(err, markErr)
		}
		return r.syncRoutes(ctx)
	}

	containerID, err := r.docker.StartContainer(ctx, project.ContainerName, project.ID, image, env)
	if err != nil {
		if markErr := r.projects.MarkWakeFailed(project.ID, err); markErr != nil {
			return errors.Join(err, markErr)
		}
		return r.syncRoutes(ctx)
	}

	if err := r.projects.MarkWakeRunning(project.ID, image, containerID); err != nil {
		_ = r.docker.RemoveContainer(ctx, project.ContainerName)
		return err
	}

	return r.syncRoutes(ctx)
}

func (r *Reconciler) sleepIdleProjects(ctx context.Context) {
	projects, err := r.projects.RunningAutoSleepProjects()
	if err != nil {
		r.log.Error("failed to list auto sleep projects", "err", err)
		return
	}

	now := time.Now().UTC()
	for _, project := range projects {
		if project.AutoSleepAfterMS == nil || project.LastActiveAt == nil {
			continue
		}

		idleFor := time.Duration(*project.AutoSleepAfterMS) * time.Millisecond
		// If the last request plus the configured idle window is still in the future, the project is not idle yet.
		if project.LastActiveAt.Add(idleFor).After(now) {
			continue
		}

		if err := r.sleepProject(ctx, project); err != nil {
			r.log.Error("failed to sleep project", "project_id", project.ID, "err", err)
		}
	}
}

func (r *Reconciler) sleepProject(ctx context.Context, project database.Project) error {
	if err := r.projects.MarkSleeping(project.ID); err != nil {
		if errors.Is(err, ErrProjectStateChanged) {
			return nil
		}
		return err
	}

	if err := r.syncRoutes(ctx); err != nil {
		if markErr := r.projects.MarkSleepAborted(project.ID, err); markErr != nil {
			return errors.Join(err, markErr)
		}
		return err
	}

	return r.removeSleepingContainer(ctx, project)
}

func (r *Reconciler) cleanupSleepingContainers(ctx context.Context) {
	// This catches interrupted sleep transitions where the DB reached sleeping before the container was removed.
	projects, err := r.projects.SleepingProjectsWithContainer()
	if err != nil {
		r.log.Error("failed to list sleeping projects with containers", "err", err)
		return
	}

	for _, project := range projects {
		if err := r.removeSleepingContainer(ctx, project); err != nil {
			r.log.Error("failed to remove sleeping project container", "project_id", project.ID, "err", err)
		}
	}
}

func (r *Reconciler) removeSleepingContainer(ctx context.Context, project database.Project) error {
	r.projects.clearRuntimeStatsCache(project.ID)
	if err := r.docker.RemoveContainer(ctx, project.ContainerName); err != nil {
		return err
	}

	return r.projects.MarkSlept(project.ID)
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

func (r *Reconciler) syncRoutes(ctx context.Context) error {
	if r.routes == nil {
		return nil
	}

	return r.routes.Sync(ctx)
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

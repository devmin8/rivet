package services

import (
	"context"
	"fmt"
	"time"

	"github.com/devmin8/rivet/internal/server/database"
)

func (s *ProjectService) RunProjectReconciler(ctx context.Context, interval time.Duration, stuckAfter time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	if stuckAfter <= 0 {
		stuckAfter = 10 * time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.ReconcileProjects(ctx, stuckAfter)
		}
	}
}

func (s *ProjectService) ReconcileProjects(ctx context.Context, stuckAfter time.Duration) error {
	if stuckAfter <= 0 {
		stuckAfter = 10 * time.Minute
	}

	now := time.Now().UTC()
	cutoff := now.Add(-stuckAfter)

	if err := s.db.WithContext(ctx).
		Model(&database.Project{}).
		Where("is_active = ? AND desired_status = ? AND status = ? AND status_updated_at < ?", true, database.DesiredStatusRunning, database.StatusCreating, cutoff).
		Updates(map[string]any{
			"status":            database.StatusFailed,
			"status_updated_at": now,
			"error":             "project deployment timed out",
		}).Error; err != nil {
		return err
	}

	if err := s.db.WithContext(ctx).
		Model(&database.Project{}).
		Where("is_active = ? AND desired_status = ? AND status = ? AND (container_id = ? OR container_id IS NULL)", true, database.DesiredStatusRunning, database.StatusRunning, "").
		Updates(map[string]any{
			"status":            database.StatusFailed,
			"status_updated_at": now,
			"error":             "project is marked running without a container",
		}).Error; err != nil {
		return err
	}

	if s.docker == nil {
		return nil
	}

	var projects []database.Project
	if err := s.db.WithContext(ctx).
		Where("is_active = ? AND desired_status = ? AND status = ? AND container_id <> ?", true, database.DesiredStatusRunning, database.StatusRunning, "").
		Find(&projects).Error; err != nil {
		return err
	}

	for i := range projects {
		project := &projects[i]
		info, err := s.docker.InspectContainer(ctx, project.ContainerID)
		if err != nil {
			return err
		}
		if !info.Exists {
			if err := s.markProjectFailedFromReconciler(ctx, project.ID, "project container is missing", true); err != nil {
				return err
			}
			continue
		}
		if !info.Running {
			message := fmt.Sprintf("project container is %s", info.Status)
			if info.ExitCode != 0 {
				message = fmt.Sprintf("%s with exit code %d", message, info.ExitCode)
			}
			if info.Error != "" {
				message = fmt.Sprintf("%s: %s", message, info.Error)
			}

			if err := s.markProjectFailedFromReconciler(ctx, project.ID, message, false); err != nil {
				return err
			}
		}
	}

	// TODO: sync Caddy routes after reconciler changes project runtime state.
	return nil
}

func (s *ProjectService) markProjectFailedFromReconciler(ctx context.Context, id string, message string, clearContainer bool) error {
	now := time.Now().UTC()
	updates := map[string]any{
		"status":            database.StatusFailed,
		"status_updated_at": now,
		"error":             message,
	}
	if clearContainer {
		updates["container_id"] = ""
	}

	return s.db.WithContext(ctx).
		Model(&database.Project{}).
		Where("id = ? AND is_active = ? AND desired_status = ? AND status = ?", id, true, database.DesiredStatusRunning, database.StatusRunning).
		Updates(updates).Error
}

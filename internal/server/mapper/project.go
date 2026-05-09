package mapper

import (
	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/devmin8/rivet/internal/server/database"
	"github.com/devmin8/rivet/internal/server/services"
)

func ToProjectResponse(project *database.Project) dtos.ProjectResponse {
	return dtos.ProjectResponse{
		ID:              project.ID,
		Name:            project.Name,
		Domain:          project.Domain,
		Description:     project.Description,
		Port:            project.Port,
		Platform:        string(project.Platform),
		Status:          string(project.Status),
		DesiredStatus:   string(project.DesiredStatus),
		CurrentImageRef: project.CurrentImageRef,
		TargetImageRef:  project.TargetImageRef,
		LastError:       project.LastError,
		IsActive:        project.IsActive,
		LastActiveAt:    project.LastActiveAt,
		CreatedAt:       project.CreatedAt,
		UpdatedAt:       project.UpdatedAt,
		CreatedByID:     project.CreatedByID,
		UpdatedByID:     project.UpdatedByID,
	}
}

func ToListProjectsResponse(projects []database.Project) dtos.ListProjectsResponse {
	items := make([]dtos.ProjectResponse, 0, len(projects))
	for i := range projects {
		items = append(items, ToProjectResponse(&projects[i]))
	}

	return dtos.ListProjectsResponse{Items: items}
}

func ToProjectRuntimeStatsResponse(stats services.ProjectRuntimeStatsResponse) dtos.ProjectRuntimeStatsResponse {
	items := make([]dtos.ProjectRuntimeStatsItem, 0, len(stats.Items))
	for _, item := range stats.Items {
		items = append(items, dtos.ProjectRuntimeStatsItem{
			ProjectID:              item.ProjectID,
			CPUPercent:             item.CPUPercent,
			CPUSampleWindowSeconds: item.CPUSampleWindowSeconds,
			MemoryUsageBytes:       item.MemoryUsageBytes,
			MemoryLimitBytes:       item.MemoryLimitBytes,
			MemoryPercent:          item.MemoryPercent,
			NetworkRxBytes:         item.NetworkRxBytes,
			NetworkTxBytes:         item.NetworkTxBytes,
			Pids:                   item.Pids,
		})
	}

	return dtos.ProjectRuntimeStatsResponse{
		AsOf:  stats.AsOf,
		Stale: stats.Stale,
		Items: items,
	}
}

package mapper

import (
	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/devmin8/rivet/internal/server/database"
)

func ToCreateProjectResponse(project *database.Project) dtos.CreateProjectResponse {
	return dtos.CreateProjectResponse{
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
		ContainerName:   project.ContainerName,
		ContainerID:     project.ContainerID,
		LastError:       project.LastError,
		IsActive:        project.IsActive,
		LastActiveAt:    project.LastActiveAt,
		CreatedAt:       project.CreatedAt,
		UpdatedAt:       project.UpdatedAt,
		CreatedByID:     project.CreatedByID,
		UpdatedByID:     project.UpdatedByID,
	}
}

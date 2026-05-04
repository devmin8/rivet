package mapper

import (
	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/devmin8/rivet/internal/server/database"
)

func ToCreateProjectResponse(project *database.Project) dtos.CreateProjectResponse {
	return dtos.CreateProjectResponse{
		ID:           project.ID,
		Name:         project.Name,
		Domain:       project.Domain,
		Description:  project.Description,
		Port:         project.Port,
		Image:        project.Image,
		Platform:     string(project.Platform),
		Status:       string(project.Status),
		IsActive:     project.IsActive,
		LastActiveAt: project.LastActiveAt,
		ContainerID:  project.ContainerID,
		CreatedAt:    project.CreatedAt,
		UpdatedAt:    project.UpdatedAt,
		CreatedByID:  project.CreatedByID,
		UpdatedByID:  project.UpdatedByID,
	}
}

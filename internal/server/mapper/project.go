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
		Image:           project.Image,
		Platform:        string(project.Platform),
		Status:          string(project.Status),
		DesiredStatus:   string(project.DesiredStatus),
		StatusUpdatedAt: project.StatusUpdatedAt,
		Error:           project.Error,
		IsActive:        project.IsActive,
		LastActiveAt:    project.LastActiveAt,
		ContainerID:     project.ContainerID,
		CreatedAt:       project.CreatedAt,
		UpdatedAt:       project.UpdatedAt,
		CreatedByID:     project.CreatedByID,
		UpdatedByID:     project.UpdatedByID,
	}
}

func ToListProjectsResponse(projects []database.Project) dtos.ListProjectsResponse {
	res := dtos.ListProjectsResponse{
		Projects: make([]dtos.CreateProjectResponse, 0, len(projects)),
	}
	for i := range projects {
		res.Projects = append(res.Projects, ToCreateProjectResponse(&projects[i]))
	}

	return res
}

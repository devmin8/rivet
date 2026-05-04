package mapper

import (
	"github.com/devmin8/rivet/internal/api/dtos"
	"github.com/devmin8/rivet/internal/server/database"
)

func ToCreateProjectResponse(project *database.Project) dtos.CreateProjectResponse {
	return dtos.CreateProjectResponse{
		ID:          project.ID,
		Name:        project.Name,
		Description: project.Description,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
		CreatedByID: project.CreatedByID,
		UpdatedByID: project.UpdatedByID,
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

func ToAppResponse(app *database.App) dtos.AppResponse {
	return dtos.AppResponse{
		ID:                  app.ID,
		ProjectID:           app.ProjectID,
		Name:                app.Name,
		Domain:              app.Domain,
		Description:         app.Description,
		Port:                app.Port,
		Platform:            string(app.Platform),
		Status:              string(app.Status),
		DesiredStatus:       string(app.DesiredStatus),
		StatusUpdatedAt:     app.StatusUpdatedAt,
		CurrentDeploymentID: app.CurrentDeploymentID,
		LastActiveAt:        app.LastActiveAt,
		CreatedAt:           app.CreatedAt,
		UpdatedAt:           app.UpdatedAt,
		CreatedByID:         app.CreatedByID,
		UpdatedByID:         app.UpdatedByID,
	}
}

func ToListAppsResponse(apps []database.App) dtos.ListAppsResponse {
	res := dtos.ListAppsResponse{
		Apps: make([]dtos.AppResponse, 0, len(apps)),
	}
	for i := range apps {
		res.Apps = append(res.Apps, ToAppResponse(&apps[i]))
	}

	return res
}

func ToDeploymentResponse(deployment *database.Deployment) dtos.DeploymentResponse {
	return dtos.DeploymentResponse{
		ID:              deployment.ID,
		AppID:           deployment.AppID,
		Image:           deployment.Image,
		ContainerID:     deployment.ContainerID,
		Status:          string(deployment.Status),
		StatusUpdatedAt: deployment.StatusUpdatedAt,
		Error:           deployment.Error,
		CreatedAt:       deployment.CreatedAt,
		UpdatedAt:       deployment.UpdatedAt,
		CreatedByID:     deployment.CreatedByID,
		UpdatedByID:     deployment.UpdatedByID,
	}
}

func ToListDeploymentsResponse(deployments []database.Deployment) dtos.ListDeploymentsResponse {
	res := dtos.ListDeploymentsResponse{
		Deployments: make([]dtos.DeploymentResponse, 0, len(deployments)),
	}
	for i := range deployments {
		res.Deployments = append(res.Deployments, ToDeploymentResponse(&deployments[i]))
	}

	return res
}

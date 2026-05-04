package dtos

import "time"

type CreateProjectRequest struct {
	Name        string `json:"name" validate:"required,notblank,max=255"`
	Description string `json:"description" validate:"max=2048"`
}

type CreateProjectResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedByID string    `json:"created_by_id"`
	UpdatedByID string    `json:"updated_by_id"`
}

type ListProjectsResponse struct {
	Projects []CreateProjectResponse `json:"projects"`
}

type CreateAppRequest struct {
	Name        string `json:"name" validate:"required,notblank,max=255"`
	Domain      string `json:"domain" validate:"required,domain_or_url,max=255"`
	Description string `json:"description" validate:"max=2048"`
	Port        uint32 `json:"port" validate:"required,port"`
	Platform    string `json:"platform" validate:"omitempty,oneof=linux/amd64 linux/arm64"`
}

type AppResponse struct {
	ID                  string     `json:"id"`
	ProjectID           string     `json:"project_id"`
	Name                string     `json:"name"`
	Domain              string     `json:"domain"`
	Description         string     `json:"description"`
	Port                string     `json:"port"`
	Platform            string     `json:"platform"`
	Status              string     `json:"status"`
	DesiredStatus       string     `json:"desired_status"`
	StatusUpdatedAt     time.Time  `json:"status_updated_at"`
	CurrentDeploymentID *string    `json:"current_deployment_id"`
	LastActiveAt        *time.Time `json:"last_active_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	CreatedByID         string     `json:"created_by_id"`
	UpdatedByID         string     `json:"updated_by_id"`
}

type ListAppsResponse struct {
	Apps []AppResponse `json:"apps"`
}

type CreateDeploymentRequest struct {
	Image string `json:"image" validate:"required,notblank,max=2048"`
}

type DeploymentResponse struct {
	ID              string    `json:"id"`
	AppID           string    `json:"app_id"`
	Image           string    `json:"image"`
	ContainerID     string    `json:"container_id"`
	Status          string    `json:"status"`
	StatusUpdatedAt time.Time `json:"status_updated_at"`
	Error           string    `json:"error"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	CreatedByID     string    `json:"created_by_id"`
	UpdatedByID     string    `json:"updated_by_id"`
}

type ListDeploymentsResponse struct {
	Deployments []DeploymentResponse `json:"deployments"`
}

type ImageUploadResponse struct {
	Image string `json:"image"`
}

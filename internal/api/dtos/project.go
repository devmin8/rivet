package dtos

import "time"

type CreateProjectRequest struct {
	Name        string `json:"name" validate:"required,notblank,max=255"`
	Domain      string `json:"domain" validate:"required,domain_or_url,max=255"`
	Description string `json:"description" validate:"max=2048"`
	Port        uint32 `json:"port" validate:"required,port"`
	Image       string `json:"image" validate:"max=2048"`
	Platform    string `json:"platform" validate:"omitempty,oneof=linux/amd64 linux/arm64"`
}

type CreateProjectResponse struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Domain       string     `json:"domain"`
	Description  string     `json:"description"`
	Port         string     `json:"port"`
	Image        string     `json:"image"`
	Platform     string     `json:"platform"`
	Status       string     `json:"status"`
	IsActive     bool       `json:"is_active"`
	LastActiveAt *time.Time `json:"last_active_at"`
	ContainerID  string     `json:"container_id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CreatedByID  string     `json:"created_by_id"`
	UpdatedByID  string     `json:"updated_by_id"`
}

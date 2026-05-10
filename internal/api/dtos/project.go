package dtos

import "time"

type CreateProjectRequest struct {
	Name        string `json:"name" validate:"required,notblank,max=255"`
	Domain      string `json:"domain" validate:"required,domain_or_url,max=255"`
	Description string `json:"description" validate:"max=2048"`
	Port        uint32 `json:"port" validate:"required,port"`
	Platform    string `json:"platform" validate:"omitempty,oneof=linux/amd64 linux/arm64"`
}

type UpdateProjectRuntimeSettingsRequest struct {
	AutoSleepAfterMS *int64 `json:"auto_sleep_after_ms" validate:"omitempty,min=60000"`
}

type ProjectResponse struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Domain           string     `json:"domain"`
	Description      string     `json:"description"`
	Port             string     `json:"port"`
	Platform         string     `json:"platform"`
	Status           string     `json:"status"`
	CurrentImageRef  string     `json:"current_image_ref"`
	TargetImageRef   string     `json:"target_image_ref"`
	LastError        string     `json:"last_error"`
	AutoSleepAfterMS *int64     `json:"auto_sleep_after_ms"`
	IsActive         bool       `json:"is_active"`
	LastActiveAt     *time.Time `json:"last_active_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	CreatedByID      string     `json:"created_by_id"`
	UpdatedByID      string     `json:"updated_by_id"`
}

type ListProjectsResponse struct {
	Items []ProjectResponse `json:"items"`
}

// ProjectRuntimeStatsResponse is the public response for recent per-project runtime stats.
type ProjectRuntimeStatsResponse struct {
	// AsOf is when the backend assembled this stats response.
	AsOf time.Time `json:"as_of"`
	// Items contains stats for projects that currently have available runtime data.
	Items []ProjectRuntimeStatsItem `json:"items"`
}

// ProjectRuntimeStatsItem is one project's latest available runtime stats sample.
type ProjectRuntimeStatsItem struct {
	// ProjectID is the Rivet project ID these stats belong to.
	ProjectID string `json:"project_id"`
	// CapturedAt is when this row was collected from Docker.
	CapturedAt time.Time `json:"captured_at"`
	// Stale means this row came from cache because a fresh Docker read failed.
	Stale bool `json:"stale"`
	// CPUPercent is recent container CPU usage as a percentage of host CPU capacity.
	CPUPercent float64 `json:"cpu_percent"`
	// CPUSampleWindowSeconds is the time window used to calculate CPUPercent.
	CPUSampleWindowSeconds float64 `json:"cpu_sample_window_seconds"`
	// MemoryUsageBytes is Docker CLI-style memory usage in bytes after reclaimable cache is subtracted.
	MemoryUsageBytes uint64 `json:"memory_usage_bytes"`
	// MemoryLimitBytes is the container memory limit in bytes reported by Docker.
	MemoryLimitBytes uint64 `json:"memory_limit_bytes"`
	// MemoryPercent is MemoryUsageBytes divided by MemoryLimitBytes.
	MemoryPercent float64 `json:"memory_percent"`
	// NetworkRxBytes is total bytes received across the container's network interfaces.
	NetworkRxBytes uint64 `json:"network_rx_bytes"`
	// NetworkTxBytes is total bytes transmitted across the container's network interfaces.
	NetworkTxBytes uint64 `json:"network_tx_bytes"`
	// Pids is the current number of processes and kernel threads in the container.
	Pids uint64 `json:"pids"`
}

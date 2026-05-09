export type ProjectStatus = 'running' | 'stopped' | 'deploying' | 'failed'
export type ProjectDesiredStatus = 'running' | 'stopped'
export type ProjectDisplayStatus = 'running' | 'deploying' | 'failed' | 'paused' | 'stopped'
export type ProjectAction = 'start' | 'stop' | 'delete'

export interface Project {
  id: string
  name: string
  domain: string
  description: string
  port: string
  platform: string
  status: ProjectStatus
  desired_status: ProjectDesiredStatus
  current_image_ref: string
  target_image_ref: string
  last_error: string
  is_active: boolean
  last_active_at: string | null
  created_at: string
  updated_at: string
  created_by_id: string
  updated_by_id: string
}

export interface ProjectListResponse {
  items: Project[]
}

export interface ProjectStatsResponse {
  as_of: string
  stale: boolean
  items: ProjectStats[]
}

export interface ProjectStats {
  project_id: string
  cpu_percent: number
  cpu_sample_window_seconds: number
  memory_usage_bytes: number
  memory_limit_bytes: number
  memory_percent: number
  network_rx_bytes: number
  network_tx_bytes: number
  pids: number
}

export interface ProjectListItem {
  project: Project
  status: ProjectDisplayStatus
  stats: ProjectStats | null
}

export type ProjectStatus =
  | 'starting'
  | 'running'
  | 'stopped'
  | 'deploying'
  | 'sleeping'
  | 'waking'
  | 'failed'
export type ProjectDisplayStatus = ProjectStatus
export type ProjectAction = 'start' | 'stop' | 'delete' | 'runtime-settings'

export interface Project {
  id: string
  name: string
  domain: string
  description: string
  port: string
  platform: string
  status: ProjectStatus
  current_image_ref: string
  target_image_ref: string
  last_error: string
  auto_sleep_after_ms: number | null
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
  items: ProjectStats[]
}

export interface ProjectStats {
  project_id: string
  captured_at: string
  stale: boolean
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

import { http } from '~/lib/http'
import type { Project, ProjectListResponse, ProjectStatsResponse } from '~/features/projects/types'

export interface UpdateProjectRuntimeSettingsInput {
  projectID: string
  autoSleepAfterMS: number | null
}

export function listProjects(): Promise<ProjectListResponse> {
  return http<ProjectListResponse>('/projects')
}

export function getProjectStats(): Promise<ProjectStatsResponse> {
  return http<ProjectStatsResponse>('/projects/stats')
}

export function startProject(projectID: string): Promise<Project> {
  return http<Project>(`/projects/${encodeURIComponent(projectID)}/start`, {
    method: 'POST',
  })
}

export function stopProject(projectID: string): Promise<Project> {
  return http<Project>(`/projects/${encodeURIComponent(projectID)}/stop`, {
    method: 'POST',
  })
}

export function deleteProject(projectID: string): Promise<void> {
  return http<void>(`/projects/${encodeURIComponent(projectID)}`, {
    method: 'DELETE',
  })
}

export function updateProjectRuntimeSettings({
  projectID,
  autoSleepAfterMS,
}: UpdateProjectRuntimeSettingsInput): Promise<Project> {
  return http<Project>(`/projects/${encodeURIComponent(projectID)}/runtime-settings`, {
    method: 'PATCH',
    body: JSON.stringify({
      auto_sleep_after_ms: autoSleepAfterMS,
    }),
  })
}

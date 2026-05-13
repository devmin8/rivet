import { http } from '~/lib/http'
import type {
  DeleteProjectEnvInput,
  Project,
  ProjectEnvVar,
  ProjectEnvResponse,
  ProjectListResponse,
  ProjectStatsResponse,
  UpsertProjectEnvInput,
} from '~/features/projects/types'

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

export function listProjectEnv(projectID: string): Promise<ProjectEnvResponse> {
  return http<ProjectEnvResponse>(`/projects/${encodeURIComponent(projectID)}/env`)
}

export function upsertProjectEnv({
  projectID,
  key,
  kind,
  value,
}: UpsertProjectEnvInput): Promise<ProjectEnvVar> {
  return http<ProjectEnvVar>(
    `/projects/${encodeURIComponent(projectID)}/env/${encodeURIComponent(key)}`,
    {
      method: 'PUT',
      body: JSON.stringify({ kind, value }),
    },
  )
}

export function deleteProjectEnv({ projectID, key }: DeleteProjectEnvInput): Promise<void> {
  return http<void>(
    `/projects/${encodeURIComponent(projectID)}/env/${encodeURIComponent(key)}`,
    {
      method: 'DELETE',
    },
  )
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

import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed } from 'vue'

import {
  deleteProjectEnv,
  deleteProject,
  getProjectStats,
  listProjectEnv,
  listProjects,
  startProject,
  stopProject,
  upsertProjectEnv,
  updateProjectRuntimeSettings,
} from '~/features/projects/api'
import type { Project, ProjectDisplayStatus, ProjectStats } from '~/features/projects/types'
import { projectKeys } from '~/lib/query-keys'

export function useProjects() {
  return useQuery({
    queryKey: projectKeys.activeProjects,
    queryFn: listProjects,
    staleTime: 3_000,
    // Project status is reconciler-owned operational data, so keep it fresh while the dashboard is open.
    // Later we can replace polling with stats-driven invalidation or push updates over SSE/WebSocket.
    refetchInterval: 3_000,
    refetchIntervalInBackground: false,
  })
}

export function useProjectStats() {
  return useQuery({
    queryKey: projectKeys.runtimeStats,
    queryFn: getProjectStats,
    staleTime: 3_000,
    refetchInterval: 3_000,
    refetchIntervalInBackground: false,
    placeholderData: keepPreviousData,
    retry: 1,
  })
}

export function useProjectEnv(projectID: string) {
  return useQuery({
    queryKey: projectKeys.env(projectID),
    queryFn: () => listProjectEnv(projectID),
    staleTime: 3_000,
  })
}

export function useProjectListData() {
  const projects = useProjects()
  const stats = useProjectStats()

  const statsByProjectId = computed(() => {
    const byProjectId = new Map<string, ProjectStats>()

    for (const item of stats.data.value?.items ?? []) {
      byProjectId.set(item.project_id, item)
    }

    return byProjectId
  })

  const items = computed(() =>
    (projects.data.value?.items ?? []).map((project) => ({
      project,
      status: getProjectDisplayStatus(project),
      stats: statsByProjectId.value.get(project.id) ?? null,
    })),
  )

  return {
    projects,
    stats,
    items,
    isLoadingProjects: computed(() => projects.isLoading.value && projects.data.value === undefined),
    isLoadingStats: stats.isLoading,
    isError: projects.isError,
    error: projects.error,
    statsStale: computed(() => stats.data.value?.items.some((item) => item.stale) ?? false),
  }
}

export function useStartProject() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: startProject,
    onSuccess: () => invalidateProjectList(queryClient),
  })
}

export function useStopProject() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: stopProject,
    onSuccess: () => invalidateProjectList(queryClient),
  })
}

export function useDeleteProject() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: deleteProject,
    onSuccess: () => invalidateProjectList(queryClient),
  })
}

export function useUpdateProjectRuntimeSettings() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: updateProjectRuntimeSettings,
    onSuccess: () => invalidateProjectList(queryClient),
  })
}

export function useUpsertProjectEnv(projectID: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: upsertProjectEnv,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: projectKeys.env(projectID) })
    },
  })
}

export function useDeleteProjectEnv(projectID: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: deleteProjectEnv,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: projectKeys.env(projectID) })
    },
  })
}

export function getProjectDisplayStatus(project: Project): ProjectDisplayStatus {
  return project.status
}

function invalidateProjectList(queryClient: ReturnType<typeof useQueryClient>) {
  void queryClient.invalidateQueries({ queryKey: projectKeys.activeProjects })
  void queryClient.invalidateQueries({ queryKey: projectKeys.runtimeStats })
}

import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed } from 'vue'

import {
  deleteProject,
  getProjectStats,
  listProjects,
  startProject,
  stopProject,
} from '~/features/projects/api'
import type { Project, ProjectDisplayStatus, ProjectStats } from '~/features/projects/types'
import { projectKeys } from '~/lib/query-keys'

export function useProjects() {
  return useQuery({
    queryKey: projectKeys.activeProjects,
    queryFn: listProjects,
    staleTime: 30_000,
  })
}

export function useProjectStats() {
  return useQuery({
    queryKey: projectKeys.runtimeStats,
    queryFn: getProjectStats,
    staleTime: 5_000,
    refetchInterval: 10_000,
    refetchIntervalInBackground: false,
    placeholderData: keepPreviousData,
    retry: 1,
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
    statsStale: computed(() => stats.data.value?.stale ?? false),
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

export function getProjectDisplayStatus(project: Project): ProjectDisplayStatus {
  if (project.desired_status === 'running' && project.status === 'running') {
    return 'running'
  }

  if (project.desired_status === 'stopped' && hasRuntimeHistory(project)) {
    return 'paused'
  }

  return 'stopped'
}

function hasRuntimeHistory(project: Project): boolean {
  return (
    project.current_image_ref !== '' ||
    project.target_image_ref !== '' ||
    project.container_id !== '' ||
    project.last_active_at !== null
  )
}

function invalidateProjectList(queryClient: ReturnType<typeof useQueryClient>) {
  void queryClient.invalidateQueries({ queryKey: projectKeys.activeProjects })
  void queryClient.invalidateQueries({ queryKey: projectKeys.runtimeStats })
}

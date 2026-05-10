<script setup lang="ts">
import { TriangleAlert } from 'lucide-vue-next'
import { computed, ref } from 'vue'

import ProjectEmptyState from '~/features/projects/components/ProjectEmptyState.vue'
import ProjectList from '~/features/projects/components/ProjectList.vue'
import ProjectListSkeleton from '~/features/projects/components/ProjectListSkeleton.vue'
import {
  useDeleteProject,
  useProjectListData,
  useStartProject,
  useStopProject,
  useUpdateProjectRuntimeSettings,
} from '~/features/projects/queries'
import type { ProjectAction } from '~/features/projects/types'
import { ApiError } from '~/lib/errors'

const projectList = useProjectListData()
const startProject = useStartProject()
const stopProject = useStopProject()
const deleteProject = useDeleteProject()
const updateProjectRuntimeSettings = useUpdateProjectRuntimeSettings()

const pendingActions = ref(new Map<string, ProjectAction>())

const actionError = computed(() => {
  const error =
    startProject.error.value ??
    stopProject.error.value ??
    deleteProject.error.value ??
    updateProjectRuntimeSettings.error.value
  return errorMessage(error, 'Unable to update project.')
})

function handleStart(projectId: string) {
  runProjectAction(projectId, 'start', () => startProject.mutateAsync(projectId))
}

function handleStop(projectId: string) {
  runProjectAction(projectId, 'stop', () => stopProject.mutateAsync(projectId))
}

function handleDelete(projectId: string) {
  runProjectAction(projectId, 'delete', () => deleteProject.mutateAsync(projectId))
}

function handleUpdateAutoSleep(projectId: string, autoSleepAfterMS: number | null) {
  runProjectAction(projectId, 'runtime-settings', () =>
    updateProjectRuntimeSettings.mutateAsync({
      projectID: projectId,
      autoSleepAfterMS,
    }),
  )
}

function runProjectAction(
  projectId: string,
  action: ProjectAction,
  mutate: () => Promise<unknown>,
) {
  if (pendingActions.value.has(projectId)) {
    return
  }

  startProject.reset()
  stopProject.reset()
  deleteProject.reset()
  updateProjectRuntimeSettings.reset()

  setPendingAction(projectId, action)

  void mutate()
    .catch(() => undefined)
    .finally(() => {
      clearPendingAction(projectId)
    })
}

function setPendingAction(projectId: string, action: ProjectAction) {
  pendingActions.value = new Map(pendingActions.value).set(projectId, action)
}

function clearPendingAction(projectId: string) {
  const nextPendingActions = new Map(pendingActions.value)
  nextPendingActions.delete(projectId)
  pendingActions.value = nextPendingActions
}

function errorMessage(error: unknown, fallback: string): string {
  if (error instanceof ApiError) {
    return error.message
  }

  if (error) {
    return fallback
  }

  return ''
}

function formatProjectCount(count: number): string {
  return `${count} active ${count === 1 ? 'project' : 'projects'}`
}

function formatStatsFreshness(asOf: string | undefined): string {
  if (!asOf) {
    return 'Stats pending'
  }

  return `Stats updated ${new Intl.DateTimeFormat(undefined, {
    hour: 'numeric',
    minute: '2-digit',
    second: '2-digit',
  }).format(new Date(asOf))}`
}
</script>

<template>
  <main class="bg-background min-h-dvh">
    <header class="border-b bg-background/95">
      <div class="flex h-16 w-full items-center justify-between px-6">
        <ConsoleLogo href="/" />
        <ToggleTheme />
      </div>
    </header>

    <section class="mx-auto grid w-full max-w-6xl gap-6 px-4 py-8 sm:px-6 sm:py-10">
      <div>
        <div class="space-y-1">
          <h1 class="text-2xl font-semibold tracking-tight">Projects</h1>
          <p class="text-muted-foreground text-sm">
            {{ formatProjectCount(projectList.items.value.length) }} /
            {{ formatStatsFreshness(projectList.stats.data.value?.as_of) }}
            <span
              v-if="projectList.statsStale.value"
              class="inline-flex items-center gap-1 text-amber-600 dark:text-amber-300"
              title="Some runtime stats could not be refreshed. Cached values are shown where available."
            >
              /
              <TriangleAlert class="size-3.5" aria-hidden="true" />
              stale
            </span>
          </p>
        </div>
      </div>

      <Alert v-if="actionError" variant="destructive">
        <AlertDescription>{{ actionError }}</AlertDescription>
      </Alert>

      <Alert v-if="projectList.isError.value" variant="destructive">
        <AlertDescription>
          {{ errorMessage(projectList.error.value, 'Unable to load projects.') }}
        </AlertDescription>
      </Alert>

      <ProjectListSkeleton v-if="projectList.isLoadingProjects.value" />

      <ProjectEmptyState
        v-else-if="!projectList.isError.value && projectList.items.value.length === 0"
      />

      <ProjectList
        v-else-if="!projectList.isError.value"
        :items="projectList.items.value"
        :is-loading-stats="projectList.isLoadingStats.value"
        :pending-actions="pendingActions"
        @start="handleStart"
        @stop="handleStop"
        @delete="handleDelete"
        @update-auto-sleep="handleUpdateAutoSleep"
      />
    </section>
  </main>
</template>

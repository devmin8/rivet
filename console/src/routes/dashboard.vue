<script setup lang="ts">
import { computed, ref } from 'vue'

import ProjectEmptyState from '~/features/projects/components/ProjectEmptyState.vue'
import ProjectList from '~/features/projects/components/ProjectList.vue'
import ProjectListSkeleton from '~/features/projects/components/ProjectListSkeleton.vue'
import {
  useDeleteProject,
  useProjectListData,
  useStartProject,
  useStopProject,
} from '~/features/projects/queries'
import { ApiError } from '~/lib/errors'

const projectList = useProjectListData()
const startProject = useStartProject()
const stopProject = useStopProject()
const deleteProject = useDeleteProject()

const pendingProjectId = ref<string | null>(null)
const pendingAction = ref<'start' | 'stop' | 'delete' | null>(null)

const actionError = computed(() => {
  const error = startProject.error.value ?? stopProject.error.value ?? deleteProject.error.value
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

function runProjectAction(
  projectId: string,
  action: 'start' | 'stop' | 'delete',
  mutate: () => Promise<unknown>,
) {
  startProject.reset()
  stopProject.reset()
  deleteProject.reset()

  pendingProjectId.value = projectId
  pendingAction.value = action

  void mutate()
    .catch(() => undefined)
    .finally(() => {
      pendingProjectId.value = null
      pendingAction.value = null
    })
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
            <span v-if="projectList.statsStale.value">/ stale</span>
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
        :pending-project-id="pendingProjectId"
        :pending-action="pendingAction"
        @start="handleStart"
        @stop="handleStop"
        @delete="handleDelete"
      />
    </section>
  </main>
</template>

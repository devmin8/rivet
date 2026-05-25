<script setup lang="ts">
import { Plus, Search, TriangleAlert } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import { toast } from 'vue-sonner'

import CreateProjectDialog from '~/features/projects/components/CreateProjectDialog.vue'
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

type AutoSleepUpdateCallbacks = {
  onSuccess?: () => void
  onError?: (error: unknown) => void
}

const projectList = useProjectListData()
const startProject = useStartProject()
const stopProject = useStopProject()
const deleteProject = useDeleteProject()
const updateProjectRuntimeSettings = useUpdateProjectRuntimeSettings()

const pendingActions = ref(new Map<string, ProjectAction>())
const openCreateProjectDialog = ref(false)
const searchQuery = ref('')
const statusFilter = ref<'all' | 'running' | 'sleeping'>('all')

const visibleItems = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()

  return projectList.items.value.filter((item) => {
    const matchesStatus = statusFilter.value === 'all' || item.status === statusFilter.value
    const matchesQuery =
      query === '' ||
      item.project.name.toLowerCase().includes(query) ||
      item.project.domain.toLowerCase().includes(query)

    return matchesStatus && matchesQuery
  })
})

const runningCount = computed(
  () => projectList.items.value.filter((item) => item.status === 'running').length,
)

const sleepingCount = computed(
  () => projectList.items.value.filter((item) => item.status === 'sleeping').length,
)

const totalCPU = computed(() => {
  const total = projectList.items.value.reduce(
    (sum, item) => sum + (item.stats?.cpu_percent ?? 0),
    0,
  )
  return `${trimDecimal(total, total < 10 ? 1 : 0)}%`
})

const totalMemory = computed(() => {
  const total = projectList.items.value.reduce(
    (sum, item) => sum + (item.stats?.memory_usage_bytes ?? 0),
    0,
  )
  return formatBinaryBytes(total)
})

watch(
  () => projectList.error.value,
  (error) => {
    if (error) {
      toast.error(errorMessage(error, 'Unable to load projects.'))
    }
  },
)

function handleStart(projectId: string) {
  runProjectAction(projectId, 'start', () => startProject.mutateAsync(projectId))
}

function handleStop(projectId: string) {
  runProjectAction(projectId, 'stop', () => stopProject.mutateAsync(projectId))
}

function handleDelete(projectId: string) {
  runProjectAction(projectId, 'delete', () => deleteProject.mutateAsync(projectId))
}

function handleUpdateAutoSleep(
  projectId: string,
  autoSleepAfterMS: number | null,
  callbacks?: AutoSleepUpdateCallbacks,
) {
  runProjectAction(projectId, 'runtime-settings', () =>
    updateProjectRuntimeSettings.mutateAsync({
      projectID: projectId,
      autoSleepAfterMS,
    }),
    callbacks,
  )
}

function runProjectAction(
  projectId: string,
  action: ProjectAction,
  mutate: () => Promise<unknown>,
  callbacks?: AutoSleepUpdateCallbacks,
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
    .then(() => {
      if (action === 'runtime-settings') {
        toast.success('Auto sleep settings saved.')
      }
      callbacks?.onSuccess?.()
    })
    .catch((error: unknown) => {
      toast.error(errorMessage(error, fallbackActionError(action)))
      callbacks?.onError?.(error)
    })
    .finally(() => {
      clearPendingAction(projectId)
    })
}

function fallbackActionError(action: ProjectAction): string {
  if (action === 'runtime-settings') {
    return 'Unable to update project runtime settings.'
  }

  return 'Unable to update project.'
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

function formatBinaryBytes(value: number): string {
  return formatBytes(value, ['B', 'KiB', 'MiB', 'GiB', 'TiB'])
}

function formatBytes(value: number, units: string[]): string {
  let size = value
  let unitIndex = 0

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex += 1
  }

  const precision = size >= 10 || Number.isInteger(size) ? 0 : 1
  return `${trimDecimal(size, precision)} ${units[unitIndex]}`
}

function trimDecimal(value: number, maximumFractionDigits: number): string {
  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits,
  }).format(value)
}
</script>

<template>
  <section class="min-h-dvh">
    <div class="flex min-h-14 items-center justify-between gap-4 border-b px-6 pl-14 md:pl-6">
      <div class="flex min-w-0 items-center gap-2">
        <h1 class="truncate text-[15px] font-semibold tracking-tight">Projects</h1>
        <span
          class="inline-flex items-center rounded border px-2 py-0.5 text-[11px] font-medium text-muted-foreground"
        >
          {{ formatProjectCount(projectList.items.value.length) }}
        </span>
      </div>
      <p class="hidden text-xs text-muted-foreground sm:flex sm:items-center sm:gap-1.5">
        {{ formatStatsFreshness(projectList.stats.data.value?.as_of) }}
        <span
          v-if="projectList.statsStale.value"
          class="inline-flex items-center gap-1 text-amber-600 dark:text-amber-400"
          title="Some runtime stats could not be refreshed. Cached values are shown where available."
        >
          /
          <TriangleAlert class="size-3.5" aria-hidden="true" />
          stale
        </span>
      </p>
    </div>

    <div
      class="grid grid-cols-2 divide-x divide-y divide-border/60 border-b border-border/60 bg-muted/20 sm:grid-cols-4 sm:divide-y-0 dark:bg-muted/10"
    >
      <div class="px-6 py-4">
        <div class="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
          Running
        </div>
        <div class="mt-2 text-xl font-semibold tracking-tight text-emerald-700 dark:text-emerald-300">
          {{ runningCount }}
        </div>
      </div>
      <div class="px-6 py-4">
        <div class="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
          Sleeping
        </div>
        <div class="mt-2 text-xl font-semibold tracking-tight text-amber-600 dark:text-amber-400">
          {{ sleepingCount }}
        </div>
      </div>
      <div class="px-6 py-4">
        <div class="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
          Total CPU
        </div>
        <div class="mt-2 text-xl font-semibold tracking-tight">{{ totalCPU }}</div>
      </div>
      <div class="px-6 py-4">
        <div class="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
          Total Memory
        </div>
        <div class="mt-2 text-xl font-semibold tracking-tight">{{ totalMemory }}</div>
      </div>
    </div>

    <div
      class="flex flex-col gap-3 border-b border-border/60 bg-background/95 px-6 py-3 md:flex-row md:items-center"
    >
      <div class="flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center">
        <div class="relative w-full sm:w-72 md:w-80">
          <Search
            class="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground"
            aria-hidden="true"
          />
          <Input
            v-model="searchQuery"
            type="search"
            placeholder="Filter projects..."
            class="h-9 bg-muted/30 pl-8 text-sm shadow-none focus-visible:bg-background dark:bg-muted/20"
          />
        </div>
        <Tabs v-model="statusFilter" class="w-full sm:w-auto">
          <TabsList class="w-full sm:w-auto">
            <TabsTrigger value="all" class="flex-1 sm:flex-none">All</TabsTrigger>
            <TabsTrigger value="running" class="flex-1 sm:flex-none">Running</TabsTrigger>
            <TabsTrigger value="sleeping" class="flex-1 sm:flex-none">Sleeping</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>
      <Button
        type="button"
        size="sm"
        class="bg-primary text-primary-foreground hover:bg-primary/90 md:ml-auto"
        @click="openCreateProjectDialog = true"
      >
        <Plus class="size-4" aria-hidden="true" />
        Add Project
      </Button>
    </div>

    <ProjectListSkeleton v-if="projectList.isLoadingProjects.value" />

    <ProjectEmptyState
      v-else-if="!projectList.isError.value && projectList.items.value.length === 0"
      @add-project="openCreateProjectDialog = true"
    />

    <div v-else-if="!projectList.isError.value">
      <ProjectList
        v-if="visibleItems.length > 0"
        :items="visibleItems"
        :is-loading-stats="projectList.isLoadingStats.value"
        :pending-actions="pendingActions"
        @start="handleStart"
        @stop="handleStop"
        @delete="handleDelete"
        @update-auto-sleep="handleUpdateAutoSleep"
      />

      <div v-else class="flex flex-col items-center justify-center px-6 py-16 text-center">
        <div class="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
          No results
        </div>
        <p class="mt-3 text-sm text-muted-foreground">No projects match your filter.</p>
      </div>
    </div>

    <CreateProjectDialog v-model:open="openCreateProjectDialog" />
  </section>
</template>

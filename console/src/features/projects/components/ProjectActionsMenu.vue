<script setup lang="ts">
import { LoaderCircle, MoreHorizontal, Play, Square, Trash2, TimerReset } from 'lucide-vue-next'
import { computed, ref } from 'vue'

import type { ProjectAction, ProjectDisplayStatus } from '~/features/projects/types'

const props = defineProps<{
  status: ProjectDisplayStatus
  autoSleepAfterMs: number | null
  // User-triggered action currently in flight; runtime state still comes from status.
  pendingAction: ProjectAction | null
}>()

const emit = defineEmits<{
  start: []
  stop: []
  delete: []
  updateAutoSleep: [autoSleepAfterMS: number | null]
}>()

const isConfirmingDelete = ref(false)

const activeStatusLabels: Partial<Record<ProjectDisplayStatus, string>> = {
  deploying: 'Deploying',
  starting: 'Starting',
  waking: 'Waking',
}

const isRunning = computed(() => props.status === 'running')
const autoSleepEnabled = computed(() => props.autoSleepAfterMs !== null)

const activeStatusLabel = computed(() => activeStatusLabels[props.status] ?? '')

const disableActions = computed(
  () => props.pendingAction !== null || activeStatusLabel.value !== '',
)

const isDeleteActionPending = computed(() => props.pendingAction === 'delete')
const isRuntimeSettingsPending = computed(() => props.pendingAction === 'runtime-settings')

function actionLabel() {
  if (activeStatusLabel.value) {
    return activeStatusLabel.value
  }

  return isRunning.value ? 'Stop' : 'Start'
}

function isActionPending() {
  return (
    props.pendingAction === 'start' ||
    props.pendingAction === 'stop' ||
    activeStatusLabel.value !== ''
  )
}
</script>

<template>
  <div class="flex flex-wrap items-center justify-end gap-2">
    <Button
      type="button"
      size="sm"
      :variant="isRunning ? 'outline' : 'default'"
      :disabled="disableActions"
      @click="isRunning ? emit('stop') : emit('start')"
    >
      <LoaderCircle v-if="isActionPending()" class="size-4 animate-spin" aria-hidden="true" />
      <Square v-else-if="isRunning" class="size-4" aria-hidden="true" />
      <Play v-else class="size-4" aria-hidden="true" />
      {{ actionLabel() }}
    </Button>

    <DropdownMenu>
      <DropdownMenuTrigger as-child>
        <Button type="button" variant="ghost" size="icon-sm" aria-label="Project actions">
          <MoreHorizontal class="size-4" aria-hidden="true" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem
          :disabled="disableActions"
          @click="emit('updateAutoSleep', autoSleepEnabled ? null : 60_000)"
        >
          <LoaderCircle
            v-if="isRuntimeSettingsPending"
            class="size-4 animate-spin"
            aria-hidden="true"
          />
          <TimerReset v-else class="size-4" aria-hidden="true" />
          {{ autoSleepEnabled ? 'Disable auto sleep' : 'Enable auto sleep' }}
        </DropdownMenuItem>
        <DropdownMenuItem
          variant="destructive"
          :disabled="disableActions"
          @click="isConfirmingDelete = true"
        >
          <Trash2 class="size-4" aria-hidden="true" />
          Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  </div>

  <!-- Delete project confirmation modal -->
  <div
    v-if="isConfirmingDelete"
    class="fixed inset-0 z-60 flex items-center justify-center bg-black/45 p-4"
    role="alertdialog"
    aria-modal="true"
    aria-labelledby="delete-project-title"
  >
    <div class="bg-background w-full max-w-sm rounded-lg border p-5 shadow-lg">
      <h2 id="delete-project-title" class="text-base font-semibold">Delete project?</h2>
      <p class="text-muted-foreground mt-2 text-sm">
        This removes the project from the dashboard and stops managing its runtime state.
      </p>

      <div class="mt-5 flex justify-end gap-2">
        <Button
          type="button"
          variant="outline"
          :disabled="isDeleteActionPending"
          @click="isConfirmingDelete = false"
        >
          Cancel
        </Button>
        <Button
          type="button"
          variant="destructive"
          :disabled="isDeleteActionPending"
          @click="emit('delete')"
        >
          <LoaderCircle
            v-if="isDeleteActionPending"
            class="size-4 animate-spin"
            aria-hidden="true"
          />
          Delete
        </Button>
      </div>
    </div>
  </div>
</template>

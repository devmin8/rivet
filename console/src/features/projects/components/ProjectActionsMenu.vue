<script setup lang="ts">
import { LoaderCircle, MoreHorizontal, Play, Square, Trash2 } from 'lucide-vue-next'
import { computed, ref } from 'vue'

import type { ProjectDisplayStatus } from '~/features/projects/types'

const props = defineProps<{
  status: ProjectDisplayStatus
  isStarting: boolean
  isStopping: boolean
  isDeleting: boolean
}>()

const emit = defineEmits<{
  start: []
  stop: []
  delete: []
}>()

const isConfirmingDelete = ref(false)

const isRunning = computed(() => props.status === 'running')
const isPrimaryPending = computed(
  () => props.isStarting || props.isStopping || props.status === 'deploying',
)
const primaryLabel = computed(() => {
  if (props.status === 'deploying') {
    return 'Deploying'
  }

  return isRunning.value ? 'Stop' : 'Start'
})
</script>

<template>
  <div class="flex flex-wrap items-center justify-end gap-2">
    <Button
      type="button"
      size="sm"
      :variant="isRunning ? 'outline' : 'default'"
      :disabled="isPrimaryPending || isDeleting"
      @click="isRunning ? emit('stop') : emit('start')"
    >
      <LoaderCircle v-if="isPrimaryPending" class="size-4 animate-spin" aria-hidden="true" />
      <Square v-else-if="isRunning" class="size-4" aria-hidden="true" />
      <Play v-else class="size-4" aria-hidden="true" />
      {{ primaryLabel }}
    </Button>

    <DropdownMenu>
      <DropdownMenuTrigger as-child>
        <Button type="button" variant="ghost" size="icon-sm" aria-label="Project actions">
          <MoreHorizontal class="size-4" aria-hidden="true" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem
          variant="destructive"
          :disabled="isPrimaryPending || isDeleting"
          @click="isConfirmingDelete = true"
        >
          <Trash2 class="size-4" aria-hidden="true" />
          Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  </div>

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
          :disabled="isDeleting"
          @click="isConfirmingDelete = false"
        >
          Cancel
        </Button>
        <Button type="button" variant="destructive" :disabled="isDeleting" @click="emit('delete')">
          <LoaderCircle v-if="isDeleting" class="size-4 animate-spin" aria-hidden="true" />
          Delete
        </Button>
      </div>
    </div>
  </div>
</template>

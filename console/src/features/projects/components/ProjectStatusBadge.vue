<script setup lang="ts">
import { computed } from 'vue'

import type { ProjectDisplayStatus } from '~/features/projects/types'

const props = defineProps<{
  status: ProjectDisplayStatus
}>()

const statusLabel = computed(() => {
  const labels: Record<ProjectDisplayStatus, string> = {
    running: 'Running',
    deploying: 'Deploying',
    failed: 'Failed',
    paused: 'Paused',
    stopped: 'Stopped',
  }

  return labels[props.status]
})
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-xs font-medium"
    :class="{
      'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300':
        status === 'running',
      'border-blue-500/30 bg-blue-500/10 text-blue-700 dark:text-blue-300': status === 'deploying',
      'border-destructive/30 bg-destructive/10 text-destructive': status === 'failed',
      'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300': status === 'paused',
      'border-muted bg-muted/40 text-muted-foreground': status === 'stopped',
    }"
  >
    <span
      class="size-1.5 rounded-full"
      :class="{
        'bg-emerald-500': status === 'running',
        'bg-blue-500': status === 'deploying',
        'bg-destructive': status === 'failed',
        'bg-amber-500': status === 'paused',
        'bg-muted-foreground': status === 'stopped',
      }"
      aria-hidden="true"
    />
    {{ statusLabel }}
  </span>
</template>

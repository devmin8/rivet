<script setup lang="ts">
import { projectDisplayStatusLabels } from '~/features/projects/project-display-status'
import type { ProjectDisplayStatus } from '~/features/projects/types'

const props = defineProps<{
  status: ProjectDisplayStatus
}>()
</script>

<template>
  <span
    class="inline-flex h-6 items-center gap-1.5 rounded-full border px-2 text-[11px] font-medium leading-none"
    :class="{
      'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-400/30 dark:bg-emerald-400/15 dark:text-emerald-300':
        status === 'running',
      'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-900/60 dark:bg-sky-950/40 dark:text-sky-300':
        status === 'deploying' || status === 'starting' || status === 'waking',
      'border-destructive/30 bg-destructive/10 text-destructive': status === 'failed',
      'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900/60 dark:bg-amber-950/40 dark:text-amber-300':
        status === 'sleeping',
      'border-border bg-muted/40 text-muted-foreground': status === 'stopped',
    }"
  >
    <span
      class="size-1.5 rounded-full"
      :class="{
        'bg-emerald-500': status === 'running',
        'bg-sky-500': status === 'deploying' || status === 'starting' || status === 'waking',
        'bg-destructive': status === 'failed',
        'bg-amber-500': status === 'sleeping',
        'bg-muted-foreground': status === 'stopped',
      }"
      aria-hidden="true"
    />
    {{ projectDisplayStatusLabels[props.status] }}
  </span>
</template>

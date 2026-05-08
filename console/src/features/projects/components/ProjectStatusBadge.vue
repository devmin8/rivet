<script setup lang="ts">
import type { ProjectDisplayStatus } from '~/features/projects/types'

defineProps<{
  status: ProjectDisplayStatus
}>()
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-xs font-medium"
    :class="{
      'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300':
        status === 'running',
      'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300': status === 'paused',
      'border-muted bg-muted/40 text-muted-foreground': status === 'stopped',
    }"
  >
    <span
      class="size-1.5 rounded-full"
      :class="{
        'bg-emerald-500': status === 'running',
        'bg-amber-500': status === 'paused',
        'bg-muted-foreground': status === 'stopped',
      }"
      aria-hidden="true"
    />
    {{ status === 'running' ? 'Running' : status === 'paused' ? 'Paused' : 'Stopped' }}
  </span>
</template>

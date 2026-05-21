<script setup lang="ts">
import { TriangleAlert } from 'lucide-vue-next'
import { computed } from 'vue'

import type { ProjectDisplayStatus, ProjectStats } from '~/features/projects/types'

const props = defineProps<{
  stats: ProjectStats | null
  status: ProjectDisplayStatus
  isLoading: boolean
}>()

const emptyStatsLabel = computed(() => {
  return '-'
})

const statsTitle = computed(() => {
  if (!props.stats) {
    return undefined
  }

  const capturedAt = new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'medium',
  }).format(new Date(props.stats.captured_at))

  return props.stats.stale ? `Stale stats captured at ${capturedAt}` : `Captured at ${capturedAt}`
})

function formatCPU(value: number): string {
  if (value === 0) {
    return '0%'
  }

  if (value < 10) {
    return `${trimDecimal(value, 1)}%`
  }

  return `${trimDecimal(value, 0)}%`
}

function formatBinaryBytes(value: number): string {
  return formatBytes(value, ['B', 'KiB', 'MiB', 'GiB', 'TiB'])
}

function formatNetworkBytes(value: number): string {
  if (value < 1024) {
    return `${value} B`
  }

  return formatBytes(value, ['B', 'KiB', 'MiB', 'GiB'])
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
  <div
    class="flex min-w-0 flex-wrap items-center gap-x-6 gap-y-2 text-muted-foreground"
    :title="statsTitle"
  >
    <template v-if="stats">
      <div class="inline-flex min-w-0 items-baseline gap-2 whitespace-nowrap">
        <span class="inline-flex items-center gap-1 text-[11px] uppercase leading-5 tracking-wide">
          CPU
          <TriangleAlert
            v-if="stats.stale"
            class="size-3.5 text-amber-600 dark:text-amber-300"
            aria-label="Stale stats"
          />
        </span>
        <span class="text-[13px] leading-5 text-foreground">
          {{ formatCPU(stats.cpu_percent) }}
        </span>
      </div>

      <div class="inline-flex min-w-0 items-baseline gap-2">
        <span class="text-[11px] uppercase leading-5 tracking-wide">Memory</span>
        <span class="truncate text-[13px] leading-5 text-foreground">
          {{ formatBinaryBytes(stats.memory_usage_bytes) }} /
          {{ formatBinaryBytes(stats.memory_limit_bytes) }}
        </span>
      </div>

      <div class="inline-flex min-w-0 items-baseline gap-2">
        <span class="text-[11px] uppercase leading-5 tracking-wide">Network</span>
        <span class="truncate text-[13px] leading-5 text-foreground">
          ↓ {{ formatNetworkBytes(stats.network_rx_bytes) }} · ↑
          {{ formatNetworkBytes(stats.network_tx_bytes) }}
        </span>
      </div>
    </template>

    <template v-else-if="isLoading">
      <div v-for="index in 3" :key="index" class="inline-flex items-center gap-1.5">
        <div class="h-3 w-10 rounded bg-muted" />
        <div class="h-3 w-16 rounded bg-muted" />
      </div>
    </template>

    <p v-else class="text-[13px] leading-5 text-muted-foreground">
      {{ emptyStatsLabel }}
    </p>
  </div>
</template>

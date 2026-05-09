<script setup lang="ts">
import { computed } from 'vue'

import type { ProjectDisplayStatus, ProjectStats } from '~/features/projects/types'

const props = defineProps<{
  stats: ProjectStats | null
  status: ProjectDisplayStatus
  isLoading: boolean
}>()

const emptyStatsLabel = computed(() => {
  if (props.status === 'deploying') {
    return 'Stats pending'
  }

  if (props.status === 'running' || props.status === 'failed') {
    return 'Stats unavailable'
  }

  return 'No live stats'
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
    class="grid grid-cols-2 gap-3 text-sm sm:grid-cols-3 lg:grid-cols-[4rem_minmax(7.5rem,8rem)_minmax(6.5rem,1fr)] lg:gap-4"
  >
    <template v-if="stats">
      <div class="min-w-0">
        <p class="text-muted-foreground text-xs">CPU</p>
        <p class="font-medium">{{ formatCPU(stats.cpu_percent) }}</p>
      </div>

      <div class="min-w-0">
        <p class="text-muted-foreground text-xs">Memory</p>
        <p class="truncate text-sm font-medium">
          {{ formatBinaryBytes(stats.memory_usage_bytes) }} /
          {{ formatBinaryBytes(stats.memory_limit_bytes) }}
        </p>
        <p class="text-muted-foreground text-xs">{{ formatCPU(stats.memory_percent) }}</p>
      </div>

      <div class="col-span-2 min-w-0 sm:col-span-1">
        <p class="text-muted-foreground text-xs">Network</p>
        <p class="truncate text-sm font-medium">
          ↓ {{ formatNetworkBytes(stats.network_rx_bytes) }} ↑
          {{ formatNetworkBytes(stats.network_tx_bytes) }}
        </p>
      </div>
    </template>

    <template v-else-if="isLoading">
      <div v-for="index in 3" :key="index" class="min-w-0 space-y-2">
        <div class="bg-muted h-3 w-12 rounded" />
        <div class="bg-muted h-4 w-20 rounded" />
      </div>
    </template>

    <p v-else class="text-muted-foreground col-span-2 text-sm sm:col-span-3">
      {{ emptyStatsLabel }}
    </p>
  </div>
</template>

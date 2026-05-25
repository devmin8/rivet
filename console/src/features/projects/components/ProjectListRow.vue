<script setup lang="ts">
import { ExternalLink, TriangleAlert } from 'lucide-vue-next'
import { computed, ref } from 'vue'

import ProjectActionsMenu from '~/features/projects/components/ProjectActionsMenu.vue'
import ProjectEnvPanel from '~/features/projects/components/ProjectEnvPanel.vue'
import ProjectStatusBadge from '~/features/projects/components/ProjectStatusBadge.vue'
import type { ProjectAction, ProjectListItem } from '~/features/projects/types'

const props = defineProps<{
  item: ProjectListItem
  isLoadingStats: boolean
  pendingAction: ProjectAction | null
}>()

const emit = defineEmits<{
  start: [projectId: string]
  stop: [projectId: string]
  delete: [projectId: string]
  updateAutoSleep: [projectId: string, autoSleepAfterMS: number | null]
}>()

const isEnvDialogOpen = ref(false)

const domainHref = computed(() => {
  const domain = props.item.project.domain.trim()

  if (domain === '') {
    return ''
  }

  if (/^https?:\/\//i.test(domain)) {
    return domain
  }

  return `https://${domain}`
})

const imageRef = computed(() => {
  return props.item.project.current_image_ref || props.item.project.target_image_ref
})

const displayProjectId = computed(() => {
  return `${props.item.project.id.slice(0, 6)}...`
})

const displayImage = computed(() => {
  const image = imageRef.value

  if (image === '') {
    return ''
  }

  const digest = image.match(/sha256:([a-f0-9]{64})/i)
  if (digest) {
    return digest[1]?.slice(0, 7) ?? ''
  }

  const hexadecimalTokens = image.match(/[a-f0-9]{7,64}/gi)
  const shaToken = hexadecimalTokens?.find((token) => token.length >= 40)
  if (shaToken) {
    return shaToken.slice(0, 7)
  }

  const lastPathPart = image.split('/').at(-1) ?? image
  if (lastPathPart.length > 12) {
    return lastPathPart.slice(0, 7)
  }

  return lastPathPart
})

const emptyStatsLabel = computed(() => {
  return '-'
})

const cpuLabel = computed(() => {
  if (!props.item.stats) {
    return emptyStatsLabel.value
  }

  const value = props.item.stats.cpu_percent

  if (value === 0) {
    return '0%'
  }

  return `${trimDecimal(value, value < 10 ? 1 : 0)}%`
})

const memoryLabel = computed(() => {
  if (!props.item.stats) {
    return emptyStatsLabel.value
  }

  return `${formatBinaryBytes(props.item.stats.memory_usage_bytes)} / ${formatBinaryBytes(
    props.item.stats.memory_limit_bytes,
  )}`
})

const networkLabel = computed(() => {
  if (!props.item.stats) {
    return emptyStatsLabel.value
  }

  return `↓ ${formatNetworkBytes(props.item.stats.network_rx_bytes)} · ↑ ${formatNetworkBytes(
    props.item.stats.network_tx_bytes,
  )}`
})

function formatBinaryBytes(value: number): string {
  let size = value
  let unitIndex = 0
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex += 1
  }

  const precision = size >= 10 || Number.isInteger(size) ? 0 : 1
  return `${trimDecimal(size, precision)} ${units[unitIndex]}`
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
  <TableRow class="hover:bg-muted/30 dark:hover:bg-muted/20">
    <TableCell class="px-6 py-4">
      <div class="min-w-0 whitespace-normal">
        <div class="flex min-w-0 flex-col gap-0.5">
          <h2 class="truncate text-[13px] font-medium leading-6 tracking-tight">
            {{ item.project.name }}
          </h2>
          <span
            class="inline-flex min-w-0 items-center gap-1.5 text-[11px] text-muted-foreground"
          >
            <span>Id: {{ displayProjectId }}</span>
            <CopyButton :value="item.project.id" label="Copy project ID" class="-ml-0.5 size-4" />
          </span>
        </div>
      </div>
    </TableCell>

    <TableCell class="py-4">
      <ProjectStatusBadge :status="item.status" />
    </TableCell>

    <TableCell class="py-4">
      <a
        v-if="domainHref"
        :href="domainHref"
        target="_blank"
        rel="noopener noreferrer"
        class="inline-flex max-w-full items-center gap-1.5 text-[13px] leading-6 text-sky-700 hover:underline dark:text-sky-300"
        :aria-label="`Open ${item.project.domain} in a new tab`"
      >
        <span class="truncate">{{ item.project.domain }}</span>
        <ExternalLink class="size-3 shrink-0" aria-hidden="true" />
      </a>
      <span v-else class="text-[13px] text-muted-foreground">-</span>
    </TableCell>

    <TableCell class="py-4">
      <span class="text-[13px] leading-6 text-foreground tabular-nums">
        {{ item.project.port ? `:${item.project.port}` : '-' }}
      </span>
    </TableCell>

    <TableCell class="py-4">
      <span
        v-if="displayImage && imageRef"
        class="inline-flex min-w-0 items-center gap-1.5 text-[13px] leading-6 text-foreground"
      >
        <span class="truncate">{{ displayImage }}</span>
        <CopyButton :value="imageRef" label="Copy image reference" class="-ml-0.5 size-4" />
      </span>
      <span v-else class="text-[13px] text-muted-foreground">-</span>
    </TableCell>

    <TableCell class="py-4">
      <span
        class="inline-flex min-w-0 items-center gap-1.5 text-[13px] leading-6"
        :class="item.stats ? 'text-foreground' : 'text-muted-foreground'"
      >
        <TriangleAlert
          v-if="item.stats?.stale"
          class="size-3.5 shrink-0 text-amber-600 dark:text-amber-300"
          aria-label="Stale stats"
        />
        <span class="truncate">{{ cpuLabel }}</span>
      </span>
    </TableCell>

    <TableCell class="py-4">
      <span
        class="block min-w-0 truncate text-[13px] leading-6"
        :class="item.stats ? 'text-foreground' : 'text-muted-foreground'"
      >
        {{ memoryLabel }}
      </span>
    </TableCell>

    <TableCell class="py-4">
      <span
        class="block min-w-0 truncate text-[13px] leading-6"
        :class="item.stats ? 'text-foreground' : 'text-muted-foreground'"
      >
        {{ networkLabel }}
      </span>
    </TableCell>

    <TableCell class="px-6 py-4">
      <div class="flex shrink-0 items-center justify-end gap-1.5">
        <Dialog v-model:open="isEnvDialogOpen">
          <DialogContent class="max-h-[min(42rem,calc(100vh-2rem))] overflow-y-auto sm:max-w-2xl">
            <DialogHeader>
              <DialogTitle>Runtime environment</DialogTitle>
              <DialogDescription>
                Values are applied when {{ item.project.name }} is recreated.
              </DialogDescription>
            </DialogHeader>
            <ProjectEnvPanel :project="item.project" />
          </DialogContent>
        </Dialog>
        <ProjectActionsMenu
          :status="item.status"
          :auto-sleep-after-ms="item.project.auto_sleep_after_ms"
          :pending-action="pendingAction"
          @start="emit('start', item.project.id)"
          @stop="emit('stop', item.project.id)"
          @delete="emit('delete', item.project.id)"
          @manage-environment="isEnvDialogOpen = true"
          @update-auto-sleep="emit('updateAutoSleep', item.project.id, $event)"
        />
      </div>
    </TableCell>
  </TableRow>
</template>

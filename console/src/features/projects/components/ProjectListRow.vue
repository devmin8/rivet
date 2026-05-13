<script setup lang="ts">
import { ExternalLink, SlidersHorizontal } from 'lucide-vue-next'
import { computed } from 'vue'

import ProjectActionsMenu from '~/features/projects/components/ProjectActionsMenu.vue'
import ProjectEnvPanel from '~/features/projects/components/ProjectEnvPanel.vue'
import ProjectStatsInline from '~/features/projects/components/ProjectStatsInline.vue'
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

const displayImage = computed(() => {
  // todo: think of a better way to handle this and show. for now its fine because we want
  // to know whats running or whats about to run.
  const image = props.item.project.current_image_ref || props.item.project.target_image_ref

  if (image === '') {
    return ''
  }

  const digestIndex = image.indexOf('@sha256:')
  if (digestIndex !== -1) {
    return `sha256:${image.slice(digestIndex + '@sha256:'.length, digestIndex + '@sha256:'.length + 7)}`
  }

  const lastPathPart = image.split('/').at(-1)
  return lastPathPart ?? image
})
</script>

<template>
  <article class="overflow-hidden rounded-lg border transition-colors hover:bg-muted/30">
    <div
      class="grid gap-4 p-4 lg:grid-cols-[minmax(13rem,1.35fr)_minmax(10rem,0.9fr)_minmax(8rem,0.75fr)_4.5rem_minmax(18rem,20rem)_max-content] lg:items-center xl:grid-cols-[minmax(14rem,1.45fr)_minmax(12rem,1fr)_minmax(10rem,0.9fr)_4.5rem_minmax(21rem,22rem)_max-content]"
    >
      <div class="min-w-0 space-y-1">
        <div class="flex min-w-0 flex-wrap items-center gap-2">
          <h2 class="truncate text-sm font-semibold">{{ item.project.name }}</h2>
          <ProjectStatusBadge :status="item.status" />
        </div>
        <p class="text-muted-foreground line-clamp-1 text-sm">
          {{ item.project.description || 'No description' }}
        </p>
      </div>

      <div class="min-w-0">
        <p class="text-muted-foreground text-xs lg:hidden">Domain</p>
        <a
          v-if="domainHref"
          :href="domainHref"
          target="_blank"
          rel="noopener noreferrer"
          class="text-primary inline-flex max-w-full items-center gap-1 truncate text-sm hover:underline"
        >
          <span class="truncate">{{ item.project.domain }}</span>
          <ExternalLink class="size-3.5 shrink-0" aria-hidden="true" />
        </a>
        <span v-else class="text-muted-foreground text-sm">No domain</span>
      </div>

      <div class="min-w-0">
        <p class="text-muted-foreground text-xs lg:hidden">Image</p>
        <p v-if="displayImage" class="truncate font-mono text-xs">{{ displayImage }}</p>
        <p v-else class="text-muted-foreground text-sm">No image</p>
      </div>

      <div class="min-w-0">
        <p class="text-muted-foreground text-xs lg:hidden">Port</p>
        <p v-if="item.project.port" class="font-mono text-sm">:{{ item.project.port }}</p>
        <p v-else class="text-muted-foreground text-sm">No port</p>
      </div>

      <ProjectStatsInline
        :stats="item.stats"
        :status="item.status"
        :is-loading="isLoadingStats && item.stats === null"
      />

      <div class="flex gap-2 lg:justify-self-end">
        <Dialog>
          <DialogTrigger as-child>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              title="Manage environment"
            >
              <SlidersHorizontal class="size-4" aria-hidden="true" />
            </Button>
          </DialogTrigger>
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
          @update-auto-sleep="emit('updateAutoSleep', item.project.id, $event)"
        />
      </div>
    </div>
  </article>
</template>

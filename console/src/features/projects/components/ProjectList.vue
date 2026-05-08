<script setup lang="ts">
import ProjectListRow from '~/features/projects/components/ProjectListRow.vue'
import type { ProjectAction, ProjectListItem } from '~/features/projects/types'

defineProps<{
  items: ProjectListItem[]
  isLoadingStats: boolean
  pendingActions: ReadonlyMap<string, ProjectAction>
}>()

const emit = defineEmits<{
  start: [projectId: string]
  stop: [projectId: string]
  delete: [projectId: string]
}>()
</script>

<template>
  <div class="space-y-3">
    <ProjectListRow
      v-for="item in items"
      :key="item.project.id"
      :item="item"
      :is-loading-stats="isLoadingStats"
      :pending-action="pendingActions.get(item.project.id) ?? null"
      @start="emit('start', $event)"
      @stop="emit('stop', $event)"
      @delete="emit('delete', $event)"
    />
  </div>
</template>

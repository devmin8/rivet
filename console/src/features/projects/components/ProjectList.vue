<script setup lang="ts">
import ProjectListRow from '~/features/projects/components/ProjectListRow.vue'
import type { ProjectListItem } from '~/features/projects/types'

defineProps<{
  items: ProjectListItem[]
  isLoadingStats: boolean
  pendingProjectId: string | null
  pendingAction: 'start' | 'stop' | 'delete' | null
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
      :pending-action="pendingProjectId === item.project.id ? pendingAction : null"
      @start="emit('start', $event)"
      @stop="emit('stop', $event)"
      @delete="emit('delete', $event)"
    />
  </div>
</template>

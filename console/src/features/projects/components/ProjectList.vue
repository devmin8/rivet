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
  updateAutoSleep: [projectId: string, autoSleepAfterMS: number | null]
}>()
</script>

<template>
  <Table class="min-w-[1080px] table-fixed">
    <colgroup>
      <col class="w-[18%]" />
      <col class="w-[8%]" />
      <col class="w-[15%]" />
      <col class="w-[6%]" />
      <col class="w-[9%]" />
      <col class="w-[7%]" />
      <col class="w-[12%]" />
      <col class="w-[12%]" />
      <col class="w-[13%]" />
    </colgroup>
    <TableHeader class="bg-muted/20 dark:bg-muted/10">
      <TableRow class="hover:bg-transparent">
        <TableHead
          scope="col"
          class="px-6 py-2.5 text-[11px] font-medium uppercase tracking-wide text-muted-foreground"
        >
          Project
        </TableHead>
        <TableHead scope="col" class="py-2.5 text-[11px] font-medium uppercase tracking-wide">
          Status
        </TableHead>
        <TableHead scope="col" class="py-2.5 text-[11px] font-medium uppercase tracking-wide">
          Domain
        </TableHead>
        <TableHead scope="col" class="py-2.5 text-[11px] font-medium uppercase tracking-wide">
          Port
        </TableHead>
        <TableHead scope="col" class="py-2.5 text-[11px] font-medium uppercase tracking-wide">
          Image
        </TableHead>
        <TableHead scope="col" class="py-2.5 text-[11px] font-medium uppercase tracking-wide">
          CPU
        </TableHead>
        <TableHead scope="col" class="py-2.5 text-[11px] font-medium uppercase tracking-wide">
          Memory
        </TableHead>
        <TableHead scope="col" class="py-2.5 text-[11px] font-medium uppercase tracking-wide">
          Network
        </TableHead>
        <TableHead
          scope="col"
          class="px-6 py-2.5 text-right text-[11px] font-medium uppercase tracking-wide"
        >
          Actions
        </TableHead>
      </TableRow>
    </TableHeader>
    <TableBody>
      <ProjectListRow
        v-for="item in items"
        :key="item.project.id"
        :item="item"
        :is-loading-stats="isLoadingStats"
        :pending-action="pendingActions.get(item.project.id) ?? null"
        @start="emit('start', $event)"
        @stop="emit('stop', $event)"
        @delete="emit('delete', $event)"
        @update-auto-sleep="
          (projectId, autoSleepAfterMS) => emit('updateAutoSleep', projectId, autoSleepAfterMS)
        "
      />
    </TableBody>
  </Table>
</template>

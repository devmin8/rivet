<script setup lang="ts">
import { Plus } from 'lucide-vue-next'

const emit = defineEmits<{
  addProject: []
}>()
</script>

<template>
  <div class="flex flex-col items-center px-6 py-16 text-center">
    <h2 class="text-sm font-medium uppercase tracking-wide text-muted-foreground">
      No active projects
    </h2>
    <p class="text-muted-foreground mx-auto mt-2 max-w-md text-sm">
      Create a project from a public Docker image or ship one from the CLI.
    </p>
    <Button type="button" class="mt-5" size="sm" @click="emit('addProject')">
      <Plus class="size-4" aria-hidden="true" />
      Add Project
    </Button>
  </div>
</template>

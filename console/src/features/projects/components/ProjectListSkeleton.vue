<template>
  <div class="space-y-3" aria-label="Loading projects">
    <div v-for="index in 3" :key="index" class="grid gap-4 rounded-lg border p-4 lg:grid-cols-6">
      <div class="space-y-2 lg:col-span-2">
        <div class="bg-muted h-4 w-40 rounded" />
        <div class="bg-muted h-3 w-full max-w-sm rounded" />
      </div>
      <div class="bg-muted h-4 w-32 rounded" />
      <div class="bg-muted h-4 w-24 rounded" />
      <div class="bg-muted h-4 w-14 rounded" />
      <div class="flex gap-2">
        <div class="bg-muted h-8 w-20 rounded" />
        <div class="bg-muted size-8 rounded" />
      </div>
    </div>
  </div>
</template>

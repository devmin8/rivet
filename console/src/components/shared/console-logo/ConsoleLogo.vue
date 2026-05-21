<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '~/lib/cn'

const props = defineProps<{
  href?: string
  class?: HTMLAttributes['class']
}>()
</script>

<template>
  <component
    :is="props.href ? 'a' : 'div'"
    :href="props.href"
    :aria-label="props.href ? 'Rivet home' : undefined"
    :class="cn('flex items-center gap-2 text-sidebar-foreground no-underline', props.class)"
  >
    <svg
      width="18"
      height="18"
      viewBox="0 0 20 20"
      fill="none"
      aria-hidden="true"
      class="shrink-0"
    >
      <rect x="2" y="13" width="16" height="4" rx="2" class="fill-sidebar-foreground opacity-25" />
      <rect x="2" y="8" width="16" height="4" rx="2" class="fill-sidebar-foreground opacity-55" />
      <rect x="2" y="3" width="16" height="4" rx="2" class="fill-sidebar-foreground" />
    </svg>
    <span class="text-sm font-semibold leading-none">Rivet</span>
  </component>
</template>

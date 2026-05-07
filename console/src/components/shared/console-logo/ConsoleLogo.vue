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
    :class="cn('flex items-center gap-3', props.class)"
  >
    <span
      class="flex size-9 items-center justify-center rounded-md bg-primary text-sm font-bold text-primary-foreground shadow-xs"
    >
      R
    </span>
    <span class="text-md font-semibold tracking-normal"> Rivet Console </span>
  </component>
</template>

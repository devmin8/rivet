<script setup lang="ts">
import type { DialogDescriptionProps } from 'reka-ui'
import { DialogDescription, useForwardProps } from 'reka-ui'

const props = defineProps<DialogDescriptionProps>()

const forwardedProps = useForwardProps(props)
</script>

<template>
  <DialogDescription data-slot="dialog-description" class="text-muted-foreground text-sm" v-bind="forwardedProps">
    <slot />
  </DialogDescription>
</template>

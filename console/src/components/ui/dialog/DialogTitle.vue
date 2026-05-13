<script setup lang="ts">
import type { DialogTitleProps } from 'reka-ui'
import { DialogTitle, useForwardProps } from 'reka-ui'

const props = defineProps<DialogTitleProps>()

const forwardedProps = useForwardProps(props)
</script>

<template>
  <DialogTitle data-slot="dialog-title" class="text-lg leading-none font-semibold" v-bind="forwardedProps">
    <slot />
  </DialogTitle>
</template>

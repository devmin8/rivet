<script setup lang="ts">
import { Check, Copy } from 'lucide-vue-next'
import { ref } from 'vue'
import type { HTMLAttributes } from 'vue'

import { cn } from '~/lib/cn'

const props = withDefaults(
  defineProps<{
    value: string
    label?: string
    class?: HTMLAttributes['class']
  }>(),
  {
    label: 'Copy',
  },
)

const copied = ref(false)
let resetCopiedTimer: ReturnType<typeof setTimeout> | undefined

async function copyValue(event: MouseEvent) {
  event.preventDefault()
  event.stopPropagation()

  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(props.value)
    } else {
      copyWithTextarea(props.value)
    }

    showCopied()
  } catch {
    copyWithTextarea(props.value)
    showCopied()
  }
}

function copyWithTextarea(value: string) {
  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'absolute'
  textarea.style.left = '-9999px'
  document.body.appendChild(textarea)
  textarea.select()
  document.execCommand('copy')
  document.body.removeChild(textarea)
}

function showCopied() {
  copied.value = true

  if (resetCopiedTimer) {
    clearTimeout(resetCopiedTimer)
  }

  resetCopiedTimer = setTimeout(() => {
    copied.value = false
  }, 900)
}
</script>

<template>
  <button
    type="button"
    :title="copied ? 'Copied' : label"
    :aria-label="copied ? 'Copied' : label"
    :class="
      cn(
        'inline-flex size-5 shrink-0 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-muted hover:text-foreground',
        props.class,
      )
    "
    @click="copyValue"
  >
    <Check v-if="copied" class="size-3.5" aria-hidden="true" />
    <Copy v-else class="size-3.5" aria-hidden="true" />
  </button>
</template>

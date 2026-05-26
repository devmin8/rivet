<script setup lang="ts">
import { useForm } from '@tanstack/vue-form'
import { LoaderCircle } from 'lucide-vue-next'
import { watch } from 'vue'
import { z } from 'zod'

import {
  autoSleepAfterMsToSeconds,
  autoSleepAfterSecondsSchema,
  autoSleepAfterSecondsToMs,
} from '~/features/projects/auto-sleep'

const autoSleepSchema = z.object({
  autoSleepAfterSeconds: autoSleepAfterSecondsSchema,
})

type AutoSleepFormValues = z.infer<typeof autoSleepSchema>

const props = defineProps<{
  autoSleepAfterMs: number | null
  isPending: boolean
}>()

const emit = defineEmits<{
  updateAutoSleep: [autoSleepAfterMS: number | null]
}>()

const form = useForm({
  defaultValues: {
    autoSleepAfterSeconds: autoSleepAfterMsToSeconds(props.autoSleepAfterMs),
  },
  validators: {
    onSubmit: autoSleepSchema,
  },
  onSubmit: ({ value }: { value: AutoSleepFormValues }) => {
    emit('updateAutoSleep', autoSleepAfterSecondsToMs(value.autoSleepAfterSeconds))
  },
})

watch(
  () => props.autoSleepAfterMs,
  (autoSleepAfterMs) => {
    form.setFieldValue('autoSleepAfterSeconds', autoSleepAfterMsToSeconds(autoSleepAfterMs))
  },
)

function isInvalid(field: { state: { meta: { isTouched: boolean; isValid: boolean } } }) {
  return field.state.meta.isTouched && !field.state.meta.isValid
}
</script>

<template>
  <form class="flex flex-col gap-3 rounded-md border p-3" @submit.prevent="form.handleSubmit">
    <form.Field name="autoSleepAfterSeconds">
      <template #default="{ field }">
        <Field :data-invalid="isInvalid(field)">
          <FieldLabel :for="`auto-sleep-${field.name}`">Sleep After (secs)</FieldLabel>
          <Input
            :id="`auto-sleep-${field.name}`"
            :name="field.name"
            :model-value="field.state.value"
            :disabled="isPending"
            type="number"
            inputmode="numeric"
            placeholder="60"
            aria-describedby="auto-sleep-description auto-sleep-error"
            :aria-invalid="isInvalid(field)"
            @blur="field.handleBlur"
            @input="field.handleChange(($event.target as HTMLInputElement).value)"
          />
          <FieldDescription id="auto-sleep-description">
            Leave empty to disable auto sleep.
          </FieldDescription>
          <FieldError
            v-if="isInvalid(field)"
            id="auto-sleep-error"
            :errors="field.state.meta.errors"
          />
        </Field>
      </template>
    </form.Field>

    <div class="flex flex-col gap-2 sm:flex-row sm:justify-end">
      <Button type="submit" :disabled="isPending">
        <LoaderCircle v-if="isPending" class="size-4 animate-spin" aria-hidden="true" />
        Save
      </Button>
    </div>
  </form>
</template>

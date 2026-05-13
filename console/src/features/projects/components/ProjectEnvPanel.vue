<script setup lang="ts">
import { useForm } from '@tanstack/vue-form'
import { Loader2, Plus, Trash2 } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { z } from 'zod'

import {
  useDeleteProjectEnv,
  useProjectEnv,
  useUpsertProjectEnv,
} from '~/features/projects/queries'
import type { Project, ProjectEnvKind } from '~/features/projects/types'
import { ApiError } from '~/lib/errors'

const envKeySchema = z
  .string()
  .trim()
  .min(1, 'Key is required.')
  .regex(
    /^[A-Za-z_][A-Za-z0-9_]*$/,
    'Keys must start with a letter or underscore and contain only letters, numbers, and underscores.',
  )
  .refine((key) => !key.startsWith('RIVET_'), 'Keys starting with RIVET_ are reserved.')

const projectEnvSchema = z.object({
  key: envKeySchema,
  kind: z.enum(['plain', 'secret']),
  value: z.string(),
})

type ProjectEnvFormValues = z.infer<typeof projectEnvSchema>

const props = defineProps<{
  project: Project
}>()

const env = useProjectEnv(props.project.id)
const upsertEnv = useUpsertProjectEnv(props.project.id)
const deleteEnv = useDeleteProjectEnv(props.project.id)

const editingKey = ref<string | null>(null)
const deletingKey = ref<string | null>(null)

const form = useForm({
  defaultValues: {
    key: '',
    kind: 'plain' as ProjectEnvKind,
    value: '',
  },
  validators: {
    onSubmit: projectEnvSchema,
  },
  onSubmit: ({ value }: { value: ProjectEnvFormValues }) => {
    upsertEnv.mutate(
      {
        projectID: props.project.id,
        key: value.key.trim(),
        kind: value.kind,
        value: value.value,
      },
      {
        onSuccess: resetForm,
      },
    )
  },
})

const sortedItems = computed(() => env.data.value?.items ?? [])

const mutationError = computed(() => {
  const error = upsertEnv.error.value ?? deleteEnv.error.value
  return errorMessage(error)
})

const requiresRestart = computed(() =>
  ['running', 'sleeping', 'waking', 'deploying'].includes(props.project.status),
)

const submitLabel = computed(() => {
  if (upsertEnv.isPending.value) {
    return editingKey.value === null ? 'Saving' : 'Updating'
  }

  return editingKey.value === null ? 'Add variable' : 'Update variable'
})

function editItem(key: string, kind: ProjectEnvKind, value: string | null) {
  editingKey.value = key
  form.setFieldValue('key', key)
  form.setFieldValue('kind', kind)
  form.setFieldValue('value', kind === 'plain' ? (value ?? '') : '')
}

function resetForm() {
  editingKey.value = null
  form.reset()
}

function setKind(
  kind: ProjectEnvKind,
  field: { handleChange: (value: ProjectEnvKind) => void },
) {
  field.handleChange(kind)
}

function removeItem(key: string) {
  if (deleteEnv.isPending.value) {
    return
  }

  deletingKey.value = key
  void deleteEnv
    .mutateAsync({ projectID: props.project.id, key })
    .catch(() => undefined)
    .finally(() => {
      deletingKey.value = null
    })
}

function formatUpdatedAt(value: string): string {
  return new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  }).format(new Date(value))
}

function errorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return error.message
  }

  if (error) {
    return 'Unable to update environment variables.'
  }

  return ''
}

function isInvalid(field: { state: { meta: { isTouched: boolean; isValid: boolean } } }) {
  return field.state.meta.isTouched && !field.state.meta.isValid
}
</script>

<template>
  <section>
    <div class="flex flex-col gap-4">
      <p v-if="requiresRestart" class="text-sm text-amber-600 dark:text-amber-300">
        Restart or redeploy this project after changes.
      </p>

      <Alert v-if="env.isError.value" variant="destructive">
        <AlertDescription>{{ errorMessage(env.error.value) || 'Unable to load environment variables.' }}</AlertDescription>
      </Alert>

      <Alert v-if="mutationError" variant="destructive">
        <AlertDescription>{{ mutationError }}</AlertDescription>
      </Alert>

      <div v-if="env.isLoading.value" class="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 class="size-4 animate-spin" aria-hidden="true" />
        Loading environment variables
      </div>

      <div v-else-if="!env.isError.value && sortedItems.length === 0" class="rounded-md border border-dashed p-4 text-sm text-muted-foreground">
        No runtime environment variables yet.
      </div>

      <div v-else-if="!env.isError.value" class="overflow-hidden rounded-md border">
        <div
          v-for="item in sortedItems"
          :key="item.key"
          class="flex flex-col gap-3 border-b p-3 last:border-b-0 sm:flex-row sm:items-center sm:justify-between"
        >
          <div class="min-w-0">
            <div class="flex min-w-0 flex-wrap items-center gap-2">
              <span class="truncate font-mono text-sm font-medium">{{ item.key }}</span>
              <span class="rounded border px-1.5 py-0.5 text-[0.6875rem] uppercase text-muted-foreground">
                {{ item.kind }}
              </span>
            </div>
            <p class="mt-1 truncate font-mono text-xs text-muted-foreground">
              <template v-if="item.kind === 'plain'">{{ item.value }}</template>
              <template v-else>{{ item.has_value ? 'Value saved' : 'No value' }}</template>
              <span class="font-sans"> / updated {{ formatUpdatedAt(item.updated_at) }}</span>
            </p>
          </div>

          <div class="flex shrink-0 gap-2">
            <Button type="button" variant="outline" size="sm" @click="editItem(item.key, item.kind, item.value)">
              {{ item.kind === 'secret' ? 'Replace' : 'Edit' }}
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              :disabled="deleteEnv.isPending.value"
              :title="`Delete ${item.key}`"
              @click="removeItem(item.key)"
            >
              <Loader2
                v-if="deleteEnv.isPending.value && deletingKey === item.key"
                class="size-4 animate-spin"
                aria-hidden="true"
              />
              <Trash2 v-else class="size-4" aria-hidden="true" />
            </Button>
          </div>
        </div>
      </div>

      <form class="flex flex-col gap-3 rounded-md border p-3" @submit.prevent="form.handleSubmit">
        <FieldGroup class="space-y-3">
          <div class="flex flex-col gap-3 md:flex-row">
            <form.Field name="key">
              <template #default="{ field }">
                <Field class="min-w-0 flex-1" :data-invalid="isInvalid(field)">
                  <FieldLabel :for="`env-key-${project.id}`">Key</FieldLabel>
                  <Input
                    :id="`env-key-${project.id}`"
                    :name="field.name"
                    :model-value="field.state.value"
                    :disabled="editingKey !== null || upsertEnv.isPending.value"
                    placeholder="DATABASE_URL"
                    autocomplete="off"
                    autocapitalize="none"
                    spellcheck="false"
                    inputmode="text"
                    aria-describedby="env-key-error"
                    :aria-invalid="isInvalid(field)"
                    @blur="field.handleBlur"
                    @input="field.handleChange(($event.target as HTMLInputElement).value)"
                  />
                  <FieldError
                    v-if="isInvalid(field)"
                    id="env-key-error"
                    :errors="field.state.meta.errors"
                  />
                </Field>
              </template>
            </form.Field>

            <form.Field name="kind">
              <template #default="{ field }">
                <Field class="md:w-48" :data-invalid="isInvalid(field)">
                  <FieldLabel>Kind</FieldLabel>
                  <div class="flex rounded-md border p-1">
                    <Button
                      type="button"
                      class="flex-1"
                      :variant="field.state.value === 'plain' ? 'default' : 'ghost'"
                      size="sm"
                      :disabled="upsertEnv.isPending.value"
                      @blur="field.handleBlur"
                      @click="setKind('plain', field)"
                    >
                      Plain
                    </Button>
                    <Button
                      type="button"
                      class="flex-1"
                      :variant="field.state.value === 'secret' ? 'default' : 'ghost'"
                      size="sm"
                      :disabled="upsertEnv.isPending.value"
                      @blur="field.handleBlur"
                      @click="setKind('secret', field)"
                    >
                      Secret
                    </Button>
                  </div>
                  <FieldError v-if="isInvalid(field)" :errors="field.state.meta.errors" />
                </Field>
              </template>
            </form.Field>
          </div>

          <form.Field name="value">
            <template #default="{ field }">
              <Field :data-invalid="isInvalid(field)">
                <FieldLabel :for="`env-value-${project.id}`">Value</FieldLabel>
                <Input
                  :id="`env-value-${project.id}`"
                  :name="field.name"
                  :model-value="field.state.value"
                  :type="form.state.values.kind === 'secret' ? 'password' : 'text'"
                  :disabled="upsertEnv.isPending.value"
                  placeholder="Value"
                  autocomplete="off"
                  aria-describedby="env-value-error"
                  :aria-invalid="isInvalid(field)"
                  @blur="field.handleBlur"
                  @input="field.handleChange(($event.target as HTMLInputElement).value)"
                />
                <FieldError
                  v-if="isInvalid(field)"
                  id="env-value-error"
                  :errors="field.state.meta.errors"
                />
              </Field>
            </template>
          </form.Field>
        </FieldGroup>

        <div class="flex flex-col gap-2 sm:flex-row sm:justify-end">
          <Button v-if="editingKey !== null" type="button" variant="ghost" @click="resetForm">
            Cancel
          </Button>
          <Button type="submit" :disabled="upsertEnv.isPending.value">
            <Loader2
              v-if="upsertEnv.isPending.value"
              class="mr-2 size-4 animate-spin"
              aria-hidden="true"
            />
            <Plus v-else class="mr-2 size-4" aria-hidden="true" />
            {{ submitLabel }}
          </Button>
        </div>
      </form>
    </div>
  </section>
</template>

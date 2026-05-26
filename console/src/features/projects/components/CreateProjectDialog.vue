<script setup lang="ts">
import { useForm } from '@tanstack/vue-form'
import { Loader2, Plus } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { z } from 'zod'

import {
  autoSleepAfterSecondsSchema,
  autoSleepAfterSecondsToMs,
} from '~/features/projects/auto-sleep'
import { useCreateProject } from '~/features/projects/queries'
import type { CreateProjectInput } from '~/features/projects/types'
import { ApiError } from '~/lib/errors'

const createProjectSchema = z.object({
  name: z.string().trim().min(1, 'Name is required.').max(255, 'Name is too long.'),
  imageRef: z
    .string()
    .trim()
    .min(1, 'Image is required.')
    .max(2048, 'Image reference is too long.')
    .regex(/^\S+$/, 'Image reference cannot contain spaces.'),
  domain: z.string().trim().min(1, 'Domain is required.').max(255, 'Domain is too long.'),
  port: z
    .number()
    .int('Port must be a whole number.')
    .min(1, 'Port must be between 1 and 65535.')
    .max(65535, 'Port must be between 1 and 65535.'),
  description: z.string().max(2048, 'Description is too long.'),
  platform: z.enum(['linux/amd64', 'linux/arm64']),
  autoSleepAfterSeconds: autoSleepAfterSecondsSchema,
})

type ProjectCreateFormValues = z.infer<typeof createProjectSchema>

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  'update:open': [open: boolean]
}>()

const createProject = useCreateProject()

const defaultValues: ProjectCreateFormValues = {
  name: '',
  imageRef: '',
  domain: '',
  port: 80,
  description: '',
  platform: 'linux/amd64',
  autoSleepAfterSeconds: '60',
}

const form = useForm({
  defaultValues,
  validators: {
    onSubmit: createProjectSchema,
  },
  onSubmit: ({ value }: { value: ProjectCreateFormValues }) => {
    createProject.mutate(toCreateInput(value), {
      onSuccess: () => {
        form.reset()
        emit('update:open', false)
      },
      onError: (error: unknown) => {
        toast.error(errorMessage(error))
      },
    })
  },
})

function toCreateInput(value: ProjectCreateFormValues): CreateProjectInput {
  return {
    name: value.name.trim(),
    imageRef: value.imageRef.trim(),
    domain: value.domain.trim(),
    port: value.port,
    description: value.description.trim(),
    platform: value.platform,
    autoSleepAfterMS: autoSleepAfterSecondsToMs(value.autoSleepAfterSeconds),
    start: true,
  }
}

function setPlatform(
  platform: CreateProjectInput['platform'],
  field: { handleChange: (value: CreateProjectInput['platform']) => void },
) {
  field.handleChange(platform)
}

function handleOpenChange(nextOpen: boolean) {
  emit('update:open', nextOpen)
  if (nextOpen) {
    createProject.reset()
    return
  }

  form.reset()
}

function errorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return error.message
  }

  if (error) {
    return 'Unable to create project.'
  }

  return ''
}

function isInvalid(field: { state: { meta: { isTouched: boolean; isValid: boolean } } }) {
  return field.state.meta.isTouched && !field.state.meta.isValid
}
</script>

<template>
  <Dialog :open="props.open" @update:open="handleOpenChange">
    <DialogContent class="flex max-h-[min(47rem,calc(100dvh-2rem))] max-w-2xl flex-col overflow-hidden">
      <DialogHeader>
        <DialogTitle>Add project</DialogTitle>
        <DialogDescription>
          Create and start a project from a public Docker image.
        </DialogDescription>
      </DialogHeader>

      <form class="flex min-h-0 flex-col gap-4" @submit.prevent="form.handleSubmit">
        <FieldGroup class="min-h-0 space-y-4 overflow-y-auto pr-1">
          <div class="flex flex-col gap-4 sm:flex-row">
            <form.Field name="name">
              <template #default="{ field }">
                <Field class="min-w-0 flex-1" :data-invalid="isInvalid(field)">
                  <FieldLabel for="project-name">Name</FieldLabel>
                  <Input
                    id="project-name"
                    :name="field.name"
                    :model-value="field.state.value"
                    :disabled="createProject.isPending.value"
                    placeholder="web"
                    autocomplete="off"
                    aria-describedby="project-name-error"
                    :aria-invalid="isInvalid(field)"
                    @blur="field.handleBlur"
                    @input="field.handleChange(($event.target as HTMLInputElement).value)"
                  />
                  <FieldError
                    v-if="isInvalid(field)"
                    id="project-name-error"
                    :errors="field.state.meta.errors"
                  />
                </Field>
              </template>
            </form.Field>

            <form.Field name="port">
              <template #default="{ field }">
                <Field class="sm:w-32" :data-invalid="isInvalid(field)">
                  <FieldLabel for="project-port">Port</FieldLabel>
                  <Input
                    id="project-port"
                    :name="field.name"
                    :model-value="String(field.state.value)"
                    :disabled="createProject.isPending.value"
                    type="number"
                    min="1"
                    max="65535"
                    inputmode="numeric"
                    aria-describedby="project-port-error"
                    :aria-invalid="isInvalid(field)"
                    @blur="field.handleBlur"
                    @input="
                      field.handleChange(Number(($event.target as HTMLInputElement).value))
                    "
                  />
                  <FieldError
                    v-if="isInvalid(field)"
                    id="project-port-error"
                    :errors="field.state.meta.errors"
                  />
                </Field>
              </template>
            </form.Field>
          </div>

          <form.Field name="imageRef">
            <template #default="{ field }">
              <Field :data-invalid="isInvalid(field)">
                <FieldLabel for="project-image">Image</FieldLabel>
                <Input
                  id="project-image"
                  :name="field.name"
                  :model-value="field.state.value"
                  :disabled="createProject.isPending.value"
                  placeholder="nginx:latest"
                  autocomplete="off"
                  autocapitalize="none"
                  spellcheck="false"
                  aria-describedby="project-image-error"
                  :aria-invalid="isInvalid(field)"
                  @blur="field.handleBlur"
                  @input="field.handleChange(($event.target as HTMLInputElement).value)"
                />
                <FieldError
                  v-if="isInvalid(field)"
                  id="project-image-error"
                  :errors="field.state.meta.errors"
                />
              </Field>
            </template>
          </form.Field>

          <form.Field name="domain">
            <template #default="{ field }">
              <Field :data-invalid="isInvalid(field)">
                <FieldLabel for="project-domain">Domain</FieldLabel>
                <Input
                  id="project-domain"
                  :name="field.name"
                  :model-value="field.state.value"
                  :disabled="createProject.isPending.value"
                  placeholder="app.example.com"
                  autocomplete="off"
                  autocapitalize="none"
                  spellcheck="false"
                  aria-describedby="project-domain-error"
                  :aria-invalid="isInvalid(field)"
                  @blur="field.handleBlur"
                  @input="field.handleChange(($event.target as HTMLInputElement).value)"
                />
                <FieldError
                  v-if="isInvalid(field)"
                  id="project-domain-error"
                  :errors="field.state.meta.errors"
                />
              </Field>
            </template>
          </form.Field>

          <form.Field name="description">
            <template #default="{ field }">
              <Field :data-invalid="isInvalid(field)">
                <FieldLabel for="project-description">Description</FieldLabel>
                <Input
                  id="project-description"
                  :name="field.name"
                  :model-value="field.state.value"
                  :disabled="createProject.isPending.value"
                  placeholder="Optional"
                  autocomplete="off"
                  aria-describedby="project-description-error"
                  :aria-invalid="isInvalid(field)"
                  @blur="field.handleBlur"
                  @input="field.handleChange(($event.target as HTMLInputElement).value)"
                />
                <FieldError
                  v-if="isInvalid(field)"
                  id="project-description-error"
                  :errors="field.state.meta.errors"
                />
              </Field>
            </template>
          </form.Field>

          <form.Field name="platform">
            <template #default="{ field }">
              <Field :data-invalid="isInvalid(field)">
                <FieldLabel>Platform</FieldLabel>
                <div class="flex rounded-md border p-1">
                  <Button
                    type="button"
                    class="flex-1"
                    :variant="field.state.value === 'linux/amd64' ? 'secondary' : 'ghost'"
                    size="sm"
                    :disabled="createProject.isPending.value"
                    @blur="field.handleBlur"
                    @click="setPlatform('linux/amd64', field)"
                  >
                    linux/amd64
                  </Button>
                  <Button
                    type="button"
                    class="flex-1"
                    :variant="field.state.value === 'linux/arm64' ? 'secondary' : 'ghost'"
                    size="sm"
                    :disabled="createProject.isPending.value"
                    @blur="field.handleBlur"
                    @click="setPlatform('linux/arm64', field)"
                  >
                    linux/arm64
                  </Button>
                </div>
                <FieldError v-if="isInvalid(field)" :errors="field.state.meta.errors" />
              </Field>
            </template>
          </form.Field>

          <form.Field name="autoSleepAfterSeconds">
            <template #default="{ field }">
              <Field :data-invalid="isInvalid(field)">
                <FieldLabel for="project-auto-sleep">Sleep After (secs)</FieldLabel>
                <Input
                  id="project-auto-sleep"
                  :name="field.name"
                  :model-value="field.state.value"
                  :disabled="createProject.isPending.value"
                  type="number"
                  inputmode="numeric"
                  placeholder="60"
                  aria-describedby="project-auto-sleep-description project-auto-sleep-error"
                  :aria-invalid="isInvalid(field)"
                  @blur="field.handleBlur"
                  @input="field.handleChange(($event.target as HTMLInputElement).value)"
                />
                <FieldDescription id="project-auto-sleep-description">
                  Leave empty to disable auto sleep.
                </FieldDescription>
                <FieldError
                  v-if="isInvalid(field)"
                  id="project-auto-sleep-error"
                  :errors="field.state.meta.errors"
                />
              </Field>
            </template>
          </form.Field>
        </FieldGroup>

        <div class="mt-4 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <Button
            type="button"
            variant="ghost"
            :disabled="createProject.isPending.value"
            @click="handleOpenChange(false)"
          >
            Cancel
          </Button>
          <Button type="submit" :disabled="createProject.isPending.value">
            <Loader2
              v-if="createProject.isPending.value"
              class="size-4 animate-spin"
              aria-hidden="true"
            />
            <Plus v-else class="size-4" aria-hidden="true" />
            {{ createProject.isPending.value ? 'Creating' : 'Create and start' }}
          </Button>
        </div>
      </form>
    </DialogContent>
  </Dialog>
</template>

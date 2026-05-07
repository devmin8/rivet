<script setup lang="ts">
import { useForm } from '@tanstack/vue-form'
import { Eye, EyeOff, LoaderCircle, LogIn } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { z } from 'zod'

import { Alert, AlertDescription } from '~/components/ui/alert'
import { Button } from '~/components/ui/button'
import { Field, FieldError, FieldGroup, FieldLabel } from '~/components/ui/field'
import { Input } from '~/components/ui/input'
import { useSignIn } from '~/features/auth/queries'
import { ApiError } from '~/lib/errors'

const signInSchema = z.object({
  username: z
    .string()
    .trim()
    .min(3, 'Username must be at least 3 characters.')
    .max(32, 'Username must be 32 characters or fewer.')
    .regex(/^[a-zA-Z0-9_-]+$/, 'Username can only use letters, numbers, underscores, and hyphens.'),
  password: z
    .string()
    .min(15, 'Password must be at least 15 characters.')
    .max(128, 'Password must be 128 characters or fewer.'),
})

type SignInFormValues = z.infer<typeof signInSchema>

const router = useRouter()
const signInMutation = useSignIn()
const showPassword = ref(false)

const form = useForm({
  defaultValues: {
    username: '',
    password: '',
  },
  validators: {
    onSubmit: signInSchema,
  },
  onSubmit: ({ value }: { value: SignInFormValues }) => {
    signInMutation.mutate(value, {
      onSuccess: () => {
        void router.replace('/')
      },
    })
  },
})

const formError = computed(() => {
  if (signInMutation.error.value instanceof ApiError) {
    return signInMutation.error.value.message
  }

  if (signInMutation.error.value) {
    return 'Unable to sign in right now.'
  }

  return ''
})

function isInvalid(field: { state: { meta: { isTouched: boolean; isValid: boolean } } }) {
  return field.state.meta.isTouched && !field.state.meta.isValid
}
</script>

<template>
  <form class="flex flex-col gap-5" autocomplete="on" @submit.prevent="form.handleSubmit">
    <FieldGroup class="space-y-5">
      <form.Field name="username">
        <template #default="{ field }">
          <Field :data-invalid="isInvalid(field)">
            <FieldLabel :for="field.name">Username</FieldLabel>

            <Input
              :id="field.name"
              :name="field.name"
              :model-value="field.state.value"
              autocomplete="username"
              autocapitalize="none"
              spellcheck="false"
              inputmode="text"
              aria-describedby="username-error"
              :aria-invalid="isInvalid(field)"
              :disabled="signInMutation.isPending.value"
              @blur="field.handleBlur"
              @input="field.handleChange(($event.target as HTMLInputElement).value)"
            />

            <FieldError
              v-if="isInvalid(field)"
              id="username-error"
              :errors="field.state.meta.errors"
            />
          </Field>
        </template>
      </form.Field>

      <form.Field name="password">
        <template #default="{ field }">
          <Field :data-invalid="isInvalid(field)">
            <FieldLabel :for="field.name">Password</FieldLabel>

            <div class="relative">
              <Input
                :id="field.name"
                :name="field.name"
                :model-value="field.state.value"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="current-password"
                aria-describedby="password-error"
                :aria-invalid="isInvalid(field)"
                :disabled="signInMutation.isPending.value"
                class="pr-10"
                @blur="field.handleBlur"
                @input="field.handleChange(($event.target as HTMLInputElement).value)"
              />

              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                class="absolute right-1 top-1/2 -translate-y-1/2"
                :aria-label="showPassword ? 'Hide password' : 'Show password'"
                :disabled="signInMutation.isPending.value"
                @click="showPassword = !showPassword"
              >
                <EyeOff v-if="showPassword" class="size-4" aria-hidden="true" />
                <Eye v-else class="size-4" aria-hidden="true" />
              </Button>
            </div>

            <FieldError
              v-if="isInvalid(field)"
              id="password-error"
              :errors="field.state.meta.errors"
            />
          </Field>
        </template>
      </form.Field>
    </FieldGroup>

    <Alert v-if="formError" variant="destructive">
      <AlertDescription>
        {{ formError }}
      </AlertDescription>
    </Alert>

    <Button type="submit" size="lg" class="w-full" :disabled="signInMutation.isPending.value">
      <LoaderCircle
        v-if="signInMutation.isPending.value"
        class="size-4 animate-spin"
        aria-hidden="true"
      />

      <LogIn v-else class="size-4" aria-hidden="true" />

      Sign in
    </Button>
  </form>
</template>

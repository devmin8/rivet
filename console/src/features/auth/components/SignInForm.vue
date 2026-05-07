<script setup lang="ts">
import { useForm } from '@tanstack/vue-form'
import { Eye, EyeOff, LoaderCircle, LogIn } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { z } from 'zod'
import { Button } from '~/components/ui/button'
import { Input } from '~/components/ui/input'
import { Label } from '~/components/ui/label'
import { useSignIn } from '~/features/auth/queries'
import { ApiError } from '~/lib/errors'

const signInSchema = z.object({
  username: z
    .string()
    .trim()
    .min(3, 'Username must be at least 3 characters.')
    .max(32, 'Username must be 32 characters or fewer.')
    .regex(
      /^[a-zA-Z0-9_-]+$/,
      'Username can only use letters, numbers, underscores, and hyphens.',
    ),
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

function getErrorMessage(error: unknown) {
  if (typeof error === 'string') {
    return error
  }

  if (
    error &&
    typeof error === 'object' &&
    'message' in error &&
    typeof error.message === 'string'
  ) {
    return error.message
  }

  return ''
}
</script>

<template>
  <form class="grid gap-5" autocomplete="on" @submit.prevent="form.handleSubmit">
    <form.Field name="username">
      <template #default="{ field }">
        <div class="grid gap-2">
          <Label :for="field.name">Username</Label>
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
          <p v-if="isInvalid(field)" id="username-error" class="text-destructive text-sm">
            {{ getErrorMessage(field.state.meta.errors[0]) }}
          </p>
        </div>
      </template>
    </form.Field>

    <form.Field name="password">
      <template #default="{ field }">
        <div class="grid gap-2">
          <Label :for="field.name">Password</Label>
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
          <p v-if="isInvalid(field)" id="password-error" class="text-destructive text-sm">
            {{ getErrorMessage(field.state.meta.errors[0]) }}
          </p>
        </div>
      </template>
    </form.Field>

    <p
      v-if="formError"
      role="alert"
      class="border-destructive/30 bg-destructive/10 text-destructive rounded-md border px-3 py-2 text-sm"
    >
      {{ formError }}
    </p>

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

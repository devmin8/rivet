<script setup lang="ts">
import SignInForm from '~/features/auth/components/SignInForm.vue'
</script>

<template>
  <main class="bg-background text-foreground min-h-dvh px-4 py-8">
    <section class="mx-auto flex min-h-dvh w-full max-w-sm flex-col justify-center">
      <Card class="rounded-md">
        <CardHeader class="space-y-2">
          <ConsoleLogo />
          <h1 class="text-3xl font-semibold tracking-normal">Sign in</h1>
        </CardHeader>

        <CardContent>
          <SignInForm />
        </CardContent>
      </Card>
    </section>
  </main>
</template>

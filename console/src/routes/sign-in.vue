<script setup lang="ts">
import SignInForm from '~/features/auth/components/SignInForm.vue'
</script>

<template>
  <main class="flex min-h-dvh flex-col justify-center px-6 py-10">
    <section class="mx-auto w-full max-w-sm">
      <div class="mb-8 flex flex-col items-center">
        <ConsoleLogo class="mb-4 [&_span]:sr-only [&_svg]:size-7" />

        <h1 class="text-xl font-semibold">Welcome to Rivet.</h1>
        <p class="text-muted-foreground mt-3 text-sm">Sign in to continue</p>
      </div>

      <SignInForm />
    </section>
  </main>
</template>

<script setup lang="ts">
import SignInForm from '~/features/auth/components/SignInForm.vue'
</script>

<template>
  <main class="bg-background text-foreground grid min-h-dvh px-4 py-8">
    <section class="mx-auto grid w-full max-w-sm content-center gap-8">
      <div class="grid gap-2">
        <p class="text-muted-foreground text-sm font-medium">Rivet Console</p>
        <h1 class="text-3xl font-semibold tracking-normal">Sign in</h1>
      </div>

      <SignInForm />
    </section>
  </main>
</template>

<script setup lang="ts">
import { ToggleTheme } from '~/components/shared/toggle-theme'
import { useCurrentUser } from '~/features/auth/queries'

const currentUser = useCurrentUser()
</script>

<template>
  <main class="bg-background min-h-dvh">
    <header class="border-b bg-background/95">
      <div class="flex h-16 w-full items-center justify-between px-6">
        <a href="/" class="flex items-center gap-3" aria-label="Rivet home">
          <span
            class="flex size-9 items-center justify-center rounded-md bg-primary text-sm font-bold text-primary-foreground shadow-xs"
          >
            R
          </span>
          <span class="text-md font-semibold tracking-normal"> Rivet Console </span>
        </a>
        <ToggleTheme />
      </div>
    </header>

    <section class="mx-auto grid w-full max-w-6xl gap-2 px-6 py-10">
      <p class="text-muted-foreground text-sm">Signed in as</p>
      <p class="font-mono text-sm">
        {{ currentUser.data.value?.id }}
      </p>
    </section>
  </main>
</template>

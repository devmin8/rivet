<template>
  <main>
    <header class="border-b bg-background/95">
      <div class="flex h-16 w-full items-center justify-between px-6">
        <a href="/" class="flex items-center gap-3" aria-label="Rivet home">
          <span
            class="flex size-9 items-center justify-center rounded-md bg-primary text-sm font-bold text-primary-foreground shadow-xs">
            R
          </span>
          <span class="text-md font-semibold tracking-normal">
            Rivet Console
          </span>
        </a>
        <ToggleTheme />
      </div>
    </header>
  </main>
</template>

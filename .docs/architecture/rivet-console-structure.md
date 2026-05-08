# Vue Frontend Structure

Use simple feature-first structure

```txt
src/
  main.ts

  routes/
    router.ts      # route table, route guards, and top-level route wiring
    dashboard.vue  # route views assemble feature data/components
    sign-in.vue

  lib/              # common utilities can go here as individual files grouped based on functionalities
    http.ts
    env.ts
    errors.ts
    query-keys.ts

  components/
    ui/        # shadcn-vue generated components
    layout/    # AppShell, Sidebar, Header
    shared/    # reusable app-specific components

  features/
    auth/
      api.ts
      queries.ts
      types.ts
      components/

    projects/
      api.ts
      queries.ts
      types.ts
      components/

  stores/
    auth.store.ts
    ui.store.ts

  composables/
    useDebounce.ts
    useConfirm.ts

  styles/
    main.css
```

## Rules

- Put route screens in `routes/*.vue`.
- Put the Vue Router setup in `routes/router.ts`.
- Route screens are assembly points: they compose feature queries, mutations, and components into a page.
- A route may combine multiple features. Do not force a route screen into a feature folder just because one feature is currently dominant.
- Keep route-specific naming product-facing, for example `routes/dashboard.vue`, `routes/sign-in.vue`, or `routes/project-detail.vue`.
- Put feature-only UI in `features/<feature>/components`.
- Put shared UI in `components/shared`.
- Keep shadcn-vue components only in `components/ui`.
- Put API calls in `features/<feature>/api.ts`.
- Put TanStack Query wrappers in `features/<feature>/queries.ts`.
- Put shared TanStack Query keys in `lib/query-keys.ts` so features can invalidate each other without importing feature internals.
- Put feature DTOs/types in `features/<feature>/types.ts`.
- Do not put route screens in `features/<feature>/pages`; feature folders should stay focused on domain capability.
- Use `computed` for derived values that are reused, expensive, or meaningfully clarify non-trivial logic. Inline simple one-off reactive expressions in templates, especially boolean prop pass-throughs like `:is-loading="status === 'loading'"`.
- Use Pinia only for client/app state, not server data.
- Use TanStack Query for all backend/server data.
- Do not create `services`, `repositories`, `models`, `controllers`, or `domain` folders.
- Add new folders only when repeated pain appears.

## Component Imports

`console/src/components` is the shared component registry. Vite uses `unplugin-vue-components` to scan that folder and auto-import matching Vue components when their PascalCase tags appear in templates.

```vue
<template>
  <Button>Save</Button>
  <ConsoleLogo />
</template>
```

Outside `console/src/components/**`, do not manually import from `~/components`. The plugin handles those imports.

```ts
// Do not do this outside src/components
import { Button } from "~/components/ui/button";
```

Feature-local components are different. Files under `features/<feature>/components` are not global, so route screens and nearby feature files should import them explicitly.

```ts
import SignInForm from "~/features/auth/components/SignInForm.vue";
```

## Component Types

The auto-import plugin writes the generated TypeScript declarations to:

```txt
console/components.d.ts
```

That file tells TypeScript and editors which global component tags exist. It is generated, but committed.

When a component is added, removed, or renamed under `console/src/components/**`, run:

```sh
bun run --cwd console components:dts
```

The pre-commit hook also checks staged changes under `console/src/components/**`, regenerates `console/components.d.ts`, and stages it before the commit.

During normal app builds, Vite runs `unplugin-vue-components` and injects the real imports. During the pre-commit type update, the script only runs the component plugin hook so it can refresh `components.d.ts` without building `dist`.

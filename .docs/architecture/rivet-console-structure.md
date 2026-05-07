# Vue Frontend Structure

Use simple feature-first structure

```txt
src/
  main.ts
  router.ts

  lib/
    http.ts
    env.ts
    errors.ts
    query-keys.ts
    utils.ts

  components/
    ui/        # shadcn-vue generated components
    layout/    # AppShell, Sidebar, Header
    shared/    # reusable app-specific components

  features/
    auth/
      api.ts
      queries.ts
      types.ts
      pages/
      components/

    projects/
      api.ts
      queries.ts
      types.ts
      pages/
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

- Put route screens in `features/<feature>/pages`.
- Put feature-only UI in `features/<feature>/components`.
- Put shared UI in `components/shared`.
- Keep shadcn-vue components only in `components/ui`.
- Put API calls in `features/<feature>/api.ts`.
- Put TanStack Query wrappers in `features/<feature>/queries.ts`.
- Put shared TanStack Query keys in `lib/query-keys.ts` so features can invalidate each other without importing feature internals.
- Put feature DTOs/types in `features/<feature>/types.ts`.
- Use Pinia only for client/app state, not server data.
- Use TanStack Query for all backend/server data.
- Do not create `services`, `repositories`, `models`, `controllers`, or `domain` folders.
- Add new folders only when repeated pain appears.

# AGENTS.md

## Project

Rivet = self-hosted PaaS.

- Backend: Go
- Frontend: Vue 3 (`./console`)
- Package manager: Bun

---

## Global

- Follow existing structure
- Keep changes minimal
- Add only necessary files
- No tests/builds unless asked
- Prefer in-place edits
- Briefly explain changes

---

## Go

- Idiomatic Go
- Prefer stdlib
- No premature abstractions
- Clear package ownership
- Pointer receivers by default

---

## Console

Stack:

- Vue 3
- TypeScript
- Composition API
- Tailwind
- shadcn-vue
- TanStack Query
- Zod

Rules:

- No raw fetch in components
- Keep API calls in `api.ts`
- TanStack Query = server state
- Pinia = client/UI state only
- No `any`
- Small focused components
- Mobile-first
- Explicit loading/error/empty states
- Prefer flex for 1D layouts; use grid only for true 2D layouts

### Components

- `console/src/components` = global registry via `unplugin-vue-components`
- Outside `console/src/components/**`, do not import from `~/components`; `unplugin-vue-components` auto-imports those components when you use their PascalCase tags
- Feature-local components must be imported explicitly
- After adding/removing/renaming components in `console/src/components/**`, run:

```bash
bun run --cwd console components:dts
```

---

## Structure

- Page = orchestration
- Feature = domain
- Component = UI section
- Composable = reusable logic
- Schema = validation

---

## Naming

- Components/folders: kebab-case
- Composables/utils: camelCase

---

## Security

- Never expose secrets
- Avoid unsafe HTML
- Treat auth/token flows carefully
- Follow OWASP guidance

---

## Architecture Docs

- Console architecture:
  `.docs/architecture/rivet-console-structure.md`

---

## Notes

- No backwards compatibility or migrations unless requested
- Pre-first-release: prefer simple current-state code over legacy support

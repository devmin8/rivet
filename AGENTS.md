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
- Explain changes briefly

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
- API calls in `api.ts`
- TanStack Query for server state
- Pinia only for client/UI state
- No `any`
- Small focused components
- Mobile-first
- Explicit loading/error/empty states
- Prefer flex for 1-dimensional layouts; use grid only when true 2D row/column layout behavior is needed

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
- Follow OWASP recommendations

---

## Architecture Docs

- Console architecture:
  `.docs/architecture/rivet-console-structure.md`

---

Note: No backwards-compatibility or migration paths unless explicitly requested; this is pre-first-release, so prefer simple current-state code over support for old local/session/data shapes.

---

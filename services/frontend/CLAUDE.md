# CLAUDE.md - Recipe App Frontend

Prioritize readability over cleverness. Follow existing patterns in the codebase before inventing new ones.

## About

React 18 + TypeScript frontend for the Recipe App. Communicates with Go backend via Vite API proxy.

## Key Directories

```
src/
├── api/           # API client
├── components/    # Reusable components (PascalCase files)
├── pages/         # Page components
├── hooks/         # Custom hooks
├── services/      # Service modules
├── types/         # TypeScript interfaces
├── styles/        # CSS (BEM convention)
├── utils/         # Helpers
├── App.tsx        # Root component
└── main.tsx       # Entry point
```

## Common Commands

```bash
npm run dev          # Vite dev server at :5173 (hot reload)
npm run build        # tsc + vite build → dist/
npm run lint         # ESLint
npm run type-check   # tsc --noEmit
```

## Standards

- **Strict TypeScript**: no implicit any, noUnusedLocals, noUnusedParameters
- **Interface over type** for object shapes
- **Functional components only** — no classes
- **PascalCase** for component files/names, **camelCase** for variables/functions
- **ComponentNameProps** for prop interfaces
- **BEM CSS**: `block__element--modifier` in `src/styles/`
- **try-catch** on all API calls, user-friendly error messages in state
- **Default export** for page/root components

## Critical Conventions

- **Always use relative API paths**: `/api/v1/...` — never hardcode backend URL
- **JSON fields are snake_case**: matches Go struct tags (`user_id`, `prep_time`, `created_at`)
- **All IDs are UUIDs**
- **JWT auth**: token from login response → `localStorage` → `Authorization: Bearer <token>` header on protected routes
- Vite proxy target: `http://app:8080` (Docker) or `http://localhost:8080` (local) — configured in `vite.config.ts`

## API Routes Reference

All under `/api/v1`. Check `services/backend/internal/router/router.go` for exact definitions.

**Public:** `POST /auth/register`, `/auth/login`, `/auth/forgot-password`, `/auth/reset-password`

**Protected (JWT):** users, ai-configs, recipes (with import/url, import/pdf, parser/instructions), shopping-lists (with nested items, toggle, add-recipe, sorted), store-chains

## Gotchas

- Backend must be running — proxy fails silently without it
- Vite proxy config in `vite.config.ts` targets `http://app:8080` (Docker hostname) — change to `http://localhost:8080` for local dev without Docker
- CORS is handled by backend middleware (`internal/middleware/cors.go`) — don't add CORS headers in frontend
- Go backend JSON uses snake_case — TypeScript interfaces must match exactly
- Shopping list routes use nested resources: `/shopping-lists/:id/items/:itemId`

## Workflow

Before making changes:
1. Read existing code to understand current patterns in the file/directory you're changing
2. For API integration: check `services/backend/internal/router/router.go` for routes and `services/backend/internal/domain/` for Go struct JSON tags
3. After changes: run `npm run type-check` and `npm run lint`
4. For new components: create CSS file in `src/styles/`, follow BEM naming

## Docker

- Dockerfile: `node:24-alpine`, port 5173
- Compose service: `frontend`, depends on `app` (backend)
- Volumes: `src/`, `vite.config.ts`, `index.html` for hot reload

## See Also

- Root project context: `../../CLAUDE.md`
- Copilot instructions: `copilot-instructions.md` (same directory, detailed code examples)
- Backend API routes: `../backend/internal/router/router.go`
- Backend domain types: `../backend/internal/domain/`

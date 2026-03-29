# CLAUDE.md - Recipe App

Prioritize readability over cleverness. Ask clarifying questions before making architecture changes.

## About This Project

Full-stack recipe management app. React 18 + TypeScript frontend, Go REST API backend, PostgreSQL database. All containerized with Docker Compose.

## Key Directories

- `services/frontend/src/` — React app (components, pages, hooks, services, types, styles)
- `services/backend/internal/` — Go backend (clean architecture: domain → handler → service → repository)
- `services/backend/migrations/` — PostgreSQL migrations (golang-migrate)
- `services/backend/internal/middleware/` — CORS (`cors.go`), JWT auth (`auth.go`)
- `services/backend/internal/router/router.go` — All API route definitions

## Common Commands

```bash
# Docker (all services)
docker-compose up                  # frontend:5173, backend:8080, db:5432

# Local development
cd services/backend && make dev    # backend at :8080
cd services/frontend && npm run dev # frontend at :5173 (proxies /api/* to backend)

# Frontend
npm run build        # tsc + vite build
npm run lint         # ESLint
npm run type-check   # tsc --noEmit

# Backend
cd services/backend && make test   # run tests
```

## Standards

### Frontend (TypeScript/React)
- Strict TypeScript — no implicit any, noUnusedLocals, noUnusedParameters
- Interface over type for object shapes
- Functional components only, PascalCase files, camelCase variables
- BEM convention for CSS (`block__element--modifier`)
- try-catch on all API calls with user-friendly error messages

### Backend (Go)
- Clean architecture: handler → service → repository
- Idiomatic Go error handling
- Gin framework, GORM for DB access

## Critical Conventions

- **API calls use relative paths only**: always `/api/v1/...`, never hardcode backend URL. Vite proxy routes to backend (`http://app:8080` in Docker, `http://localhost:8080` locally).
- **JSON fields are snake_case**: matching Go struct JSON tags (`user_id`, `prep_time`, `created_at`)
- **All IDs are UUIDs**: backend uses `uuid_generate_v4()`
- **JWT auth**: login returns token, protected routes need `Authorization: Bearer <token>` header, store in localStorage
- **Docker service name for backend is `app`**: this is the hostname used in Vite proxy config

## Gotchas

- Backend must be running for frontend to work (API proxy fails silently)
- Frontend proxy config is in `vite.config.ts` — target changes between Docker (`http://app:8080`) and local (`http://localhost:8080`)
- CORS is handled by Go middleware (`internal/middleware/cors.go`), not by Vite — don't add CORS headers in frontend
- Shopping list routes use nested resources: `/shopping-lists/:id/items/:itemId`

## Workflow

Before making changes:
1. Read the relevant existing code to understand current patterns
2. For new features: check if similar patterns exist in the codebase and follow them
3. For frontend API integration: check `services/backend/internal/router/router.go` for exact route definitions
4. For type definitions: check `services/backend/internal/domain/` for Go struct JSON tags to match field names
5. Run `npm run type-check` and `npm run lint` after frontend changes
6. Run `make test` after backend changes

## More Information

- Frontend-specific context: `services/frontend/CLAUDE.md`
- Backend-specific context: `services/backend/CLAUDE.md`
- Frontend Copilot instructions: `services/frontend/copilot-instructions.md`
- Backend docs: `services/backend/README.md`

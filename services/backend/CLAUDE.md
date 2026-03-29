# CLAUDE.md - Recipe App Backend

Prioritize readability over cleverness. Follow existing patterns in the codebase before inventing new ones. Ask clarifying questions before making architecture changes.

## About

Go REST API for recipe management. Gin framework, GORM ORM, PostgreSQL. Clean architecture with dependency injection wired in `cmd/api/main.go`.

## Key Directories

- `cmd/api/main.go` — Entry point. Wires config → DB → repos → services → handlers → router
- `internal/domain/` — Domain models and request/response types (GORM tags + JSON tags)
- `internal/handler/` — HTTP handlers, one per domain. Bind JSON → call service → return response
- `internal/service/` — Business logic. Interfaces defined here, implementations are unexported structs
- `internal/repository/` — Data access. Generic `BaseRepository[T]` with GORM. Interfaces defined here
- `internal/middleware/` — CORS (`cors.go`), JWT auth (`auth.go`), context helpers (`context.go`)
- `internal/router/router.go` — All route definitions. Public vs protected (JWT) groups
- `internal/errors/errors.go` — Custom `AppError` type with Code/Message/Err
- `pkg/` — Shared packages: `config`, `database`, `ai`, `email`, `storage`, `urlparser`, `pdfparser`
- `migrations/` — Sequential SQL migrations (golang-migrate, 13 files)

## Common Commands

```bash
make dev              # Air hot-reload dev server at :8080
make build            # Build binary to bin/
make run              # Run without hot-reload
make test             # go test -v ./...
make test-coverage    # Tests with HTML coverage report
make lint             # golangci-lint run
make fmt              # go fmt ./...

# Database
make migrate-create name=your_migration  # New migration file
make migrate-up                          # Apply all migrations
make migrate-down                        # Undo all migrations
make db-connect                          # psql into database

# Docker
make docker-up        # docker-compose up -d
make docker-down      # docker-compose down
make setup-dev        # docker-up + db-create + migrate-up
```

## Architecture Pattern

```
main.go → config → database → repositories → services → handlers → router
```

Every domain follows the same pattern:
1. **Domain model** in `internal/domain/` — struct with GORM + JSON tags
2. **Repository interface** in `internal/repository/` — embeds `Repository[T]`, adds domain methods
3. **Repository implementation** — embeds `BaseRepository[T]`, implements interface
4. **Service interface** in `internal/service/` — defines business methods
5. **Service implementation** — unexported struct, receives repo via constructor
6. **Handler** in `internal/handler/` — receives service via constructor, binds JSON, calls service
7. **Routes** in `internal/router/router.go` — wires handler methods to Gin routes

Aggregated in: `repositories.go`, `services.go`, `handlers.go`

## Standards

- Idiomatic Go error handling — return errors, don't panic
- Interfaces for repositories and services (dependency injection, testability)
- `context.Context` passed through handler → service → repository
- JSON struct tags use **snake_case** (`json:"user_id"`, `json:"prep_time"`)
- UUIDs for all primary keys (`uuid_generate_v4()`)
- `zap.Logger` for structured logging (injected via constructor)
- `gin.H{"error": ...}` for error responses
- `c.ShouldBindJSON(&req)` for request binding in handlers
- `c.Request.Context()` to pass context from handler to service

## Critical Conventions

- **Config via Viper**: loaded from `env.{APP_ENV}.yaml` (default: `env.development.yaml`)
- **Auth flow**: Register creates user + profile. Login returns JWT token. Token contains `user_id` and `email` claims.
- **Middleware sets context**: auth middleware stores `user_id` and `email` in Gin context. Use `middleware.GetUserID(c)` to retrieve.
- **CORS origins from config**: `cors.allowed_origins` in yaml → passed to `middleware.CORS()`
- **Generic base repository**: `BaseRepository[T]` provides `GetDB()`, `withDB()`, `WithTransaction()`
- **Migrations auto-run**: `database.MigrateDB()` runs on startup in `main.go`

## Gotchas

- `env.development.yaml` is gitignored — copy from `env.development.yaml.sample` for new setup
- Air hot-reload config is in `.air.toml` — builds to `tmp/main`, watches `.go` files
- `make migrate-up/down` runs inside Docker container (`docker-compose exec app`)
- `internal/errors` defines both `*AppError` (custom) and `fmt.Errorf` sentinel errors — use `AppError` for new errors
- AI model factory creates models at service init time — if API keys are missing, it logs a warning but doesn't fail
- `storage.FileStore` interface abstracts local/S3 — check `pkg/storage/` for implementations

## Workflow

Before making changes:
1. Read existing code in the domain you're changing to understand the pattern
2. For new domains: follow the User or Recipe domain as reference (domain → repo → service → handler → router)
3. For new migrations: `make migrate-create name=description` then write both up and down SQL
4. For new routes: add to `internal/router/router.go` in the correct group (public vs protected)
5. After changes: `make test` and `make lint`
6. For handler changes: verify JSON field names match domain struct tags

## Docker

- Dockerfile: `golang:1.25-alpine`, installs `migrate` CLI + Air hot-reload
- Compose service name: `app` (this is the hostname frontend uses)
- Volumes: entire `services/backend/` mounted at `/app`
- Dev command: `air` (hot-reload)
- DB depends on healthcheck before app starts

## See Also

- Root project context: `../../CLAUDE.md`
- Frontend context: `../frontend/CLAUDE.md`
- API routes: `internal/router/router.go`
- Domain models: `internal/domain/`
- Config structure: `pkg/config/config.go`

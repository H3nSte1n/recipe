---
name: 'Go Backend - General'
description: 'Architecture, conventions, and standards for the Go backend service'
applyTo: 'services/backend/**/*.go'
---

# Go Backend Standards

## Architecture

Clean architecture with dependency injection. Flow: `main.go → config → DB → repositories → services → handlers → router`.

Every domain follows the same pattern:
1. Domain model in `internal/domain/` — struct with GORM + JSON tags
2. Repository interface in `internal/repository/` — embeds `Repository[T]`, adds domain methods
3. Repository implementation — embeds `BaseRepository[T]`, implements interface
4. Service interface in `internal/service/` — defines business methods
5. Service implementation — unexported struct, receives repo via constructor
6. Handler in `internal/handler/` — receives service via constructor, binds JSON, calls service
7. Routes in `internal/router/router.go` — wires handler methods to Gin routes

Aggregated in: `repositories.go`, `services.go`, `handlers.go`

## Coding Standards

- Idiomatic Go error handling — return errors, don't panic
- Interfaces for repositories and services (dependency injection, testability)
- `context.Context` passed through handler → service → repository
- JSON struct tags use **snake_case** (`json:"user_id"`, `json:"prep_time"`)
- UUIDs for all primary keys (`uuid_generate_v4()`)
- `zap.Logger` for structured logging (injected via constructor)
- `gin.H{"error": ...}` for error responses
- `c.ShouldBindJSON(&req)` for request binding in handlers
- `c.Request.Context()` to pass context from handler to service
- Use `internal/errors.AppError` for new custom errors (not `fmt.Errorf` sentinels)

## Key Packages

- `pkg/config` — Viper-based config from `env.{APP_ENV}.yaml`
- `pkg/database` — PostgreSQL connection + auto-migration on startup
- `pkg/ai` — AI model factory (OpenAI, Anthropic)
- `pkg/storage` — FileStore interface (local + S3 implementations)
- `pkg/email` — SMTP email service
- `pkg/urlparser`, `pkg/pdfparser` — Recipe import parsers

## Auth Flow

- Register creates user + profile
- Login returns JWT token with `user_id` and `email` claims
- Auth middleware stores claims in Gin context
- Use `middleware.GetUserID(c)` to retrieve authenticated user ID
- CORS origins configured in `env.development.yaml` → `middleware.CORS()`

## Gotchas

- `env.development.yaml` is gitignored — copy from `.sample` for new setup
- `make migrate-up/down` runs inside Docker
- AI model factory logs a warning but doesn't fail if API keys are missing
- Migrations auto-run on startup via `database.MigrateDB()` in `main.go`

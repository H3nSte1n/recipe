---
name: 'Go Backend - Repositories'
description: 'Conventions for writing data access repositories in the backend'
applyTo: 'services/backend/internal/repository/**/*.go'
---

# Repository Conventions

## Pattern

- Define a **public interface** embedding `Repository[T]` and adding domain-specific methods
- Implement with a struct embedding `BaseRepository[T]`
- Constructor: `func NewXxxRepository(db *gorm.DB) XxxRepository`

## BaseRepository[T]

Generic base provides:
- `GetDB() *gorm.DB`
- `withDB(db *gorm.DB) Repository[T]` (for transactions)
- `WithTransaction(ctx context.Context, fn TransactionFunc[T]) error`

## Standards

- Accept `context.Context` as first parameter on all methods
- Use GORM query methods on `r.GetDB()`
- For transactions, use `WithTransaction` or `WithTypedTransaction` pattern
- Return `*domain.Xxx` or `error` — use GORM error types for not-found checks

## Registration

- Add new repository to `repositories.go` struct and `NewRepositories()` constructor

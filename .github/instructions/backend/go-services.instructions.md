---
name: 'Go Backend - Services'
description: 'Conventions for writing business logic services in the backend'
applyTo: 'services/backend/internal/service/**/*.go'
---

# Service Conventions

## Pattern

- Define a **public interface** with business methods
- Implement with an **unexported struct** receiving dependencies via constructor
- Constructor returns the interface type: `func NewXxxService(...) XxxService`

## Standards

- Accept `context.Context` as first parameter on all methods
- Use repository interfaces (not implementations) as dependencies
- Use `zap.Logger` for structured logging where needed
- Return domain types or custom errors — don't leak repository details
- Use `internal/errors.AppError` for domain-specific errors

## Registration

- Add new service to `services.go` struct and `NewServices()` constructor
- Services receive repositories, config, and shared packages (storage, AI, logger) via `NewServices()`

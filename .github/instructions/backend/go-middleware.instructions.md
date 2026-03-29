---
name: 'Go Backend - Middleware'
description: 'Conventions for HTTP middleware in the backend'
applyTo: 'services/backend/internal/middleware/**/*.go'
---

# Middleware Conventions

## Existing Middleware

- `cors.go` — CORS via `gin-contrib/cors`, origins from config `cors.allowed_origins`
- `auth.go` — JWT validation, stores `user_id` and `email` in Gin context
- `context.go` — Helpers to extract user info: `GetCurrentUser(c)`, `GetUserID(c)`

## Standards

- Return `gin.HandlerFunc`
- Use `c.Abort()` after writing error response to stop chain
- Use `c.Set()` / `c.Get()` for passing data through context
- New middleware is registered in `internal/router/router.go` via `engine.Use()`

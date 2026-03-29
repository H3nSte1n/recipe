---
name: 'Go Backend - Handlers'
description: 'Conventions for writing HTTP handlers in the backend'
applyTo: 'services/backend/internal/handler/**/*.go'
---

# Handler Conventions

## Pattern

Every handler follows this structure:
1. Struct with service dependency (interface type)
2. Constructor: `NewXxxHandler(service XxxService) *XxxHandler`
3. Methods receive `*gin.Context`, bind request, call service, return JSON

## Request Handling

- Bind with `c.ShouldBindJSON(&req)` — return 400 on error
- Get authenticated user via `middleware.GetUserID(c)`
- Pass `c.Request.Context()` to service methods
- Return appropriate HTTP status codes (201 for create, 200 for success, etc.)

## Error Responses

- Always use `c.JSON(status, gin.H{"error": message})` format
- Don't expose internal errors to clients — use generic messages for 500s
- Service-level errors map to 400/401/404, unexpected errors to 500

## Registration

- Add new handler to `handlers.go` struct and `NewHandlers()` constructor
- Wire handler methods to routes in `internal/router/router.go`

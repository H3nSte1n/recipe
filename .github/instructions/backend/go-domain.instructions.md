---
name: 'Go Backend - Domain Models'
description: 'Conventions for defining domain models and request/response types'
applyTo: 'services/backend/internal/domain/**/*.go'
---

# Domain Model Conventions

## Struct Tags

- GORM tags for database mapping: `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
- JSON tags in **snake_case**: `json:"user_id"`, `json:"created_at"`
- Use `json:"-"` for fields that should never be serialized (e.g., `PasswordHash`)
- Use `json:"field,omitempty"` for optional fields

## Standards

- UUIDs as string type for all primary keys
- `time.Time` for timestamps with `gorm:"autoCreateTime"` / `gorm:"autoUpdateTime"`
- Define request/response types in the same file as the domain model
- Use `validate` tags for request validation (e.g., `validate:"required,email"`)
- Pointer types for optional relationships (`*User`, `*Recipe`)
- Slice types for has-many relationships (`[]RecipeIngredient`)

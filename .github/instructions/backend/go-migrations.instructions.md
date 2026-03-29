---
name: 'Go Backend - Migrations'
description: 'Conventions for writing database migrations'
applyTo: 'services/backend/migrations/**/*.sql'
---

# Migration Conventions

## Creating Migrations

Run `make migrate-create name=description` to generate sequential up/down files.

## Standards

- Always write both `.up.sql` and `.down.sql` — down must fully reverse up
- Use `IF NOT EXISTS` / `IF EXISTS` guards where appropriate
- Use `uuid_generate_v4()` as default for UUID primary keys
- Use `TIMESTAMPTZ` for timestamp columns
- Name constraints and indexes explicitly for clean down migrations
- Seed data goes in dedicated migration files (see `000010`, `000012`)

## Running

- `make migrate-up` / `make migrate-down` — runs inside Docker container
- Migrations also auto-run on app startup via `database.MigrateDB()` in `main.go`

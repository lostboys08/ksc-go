# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Construction project management system for tracking jobs, bid line items, and payment applications. Full-stack application with Go backend, React frontend, and PostgreSQL database.

## Common Commands

```bash
# Start/stop Docker services (PostgreSQL, Adminer)
make up
make down

# Generate Go database code from SQL (run after modifying queries.sql)
make sqlc

# Run backend API server
make run-back

# Run frontend dev server
make run-front

# Reset database (destructive - wipes all data)
make reset-db
```

Adminer (database UI) available at http://localhost:8081 when Docker is running.

## Architecture

### Backend (Go)

- **SQL-first approach**: Uses SQLC for type-safe database code generation
- **Module**: `github.com/lostboys08/ksc-go/backend`
- **Structure**:
  - `cmd/api/` - API entry point
  - `internal/database/` - SQLC-generated code (models, queries)
  - `internal/service/` - Business logic layer
  - `sql/schema/` - Database migrations
  - `sql/queries.sql` - SQLC query definitions

### Database Schema

Three main tables with hierarchical job structure:
- `jobs` - Project metadata (job_number, contract_value, dates)
- `job_items` - Hierarchical bid line items using `parent_id` for tree structure
- `pay_applications` - Time-series payment tracking (monthly)

Key pattern: Recursive CTE in `GetJobTree` query fetches entire job hierarchy with depth ordering.

### Frontend (React/TypeScript)

- `src/api/` - API client layer
- `src/components/` - React components
- `src/hooks/` - Custom hooks

## SQLC Workflow

1. Define SQL queries in `backend/sql/queries.sql` with name annotations
2. Run `make sqlc` to generate Go code
3. Generated types use `github.com/google/uuid.UUID` for UUIDs
4. Generated structs include JSON tags for API serialization

## Database Connection

Local development: `postgres://ksc:password@localhost:5432/ksc_data?sslmode=disable`

ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Database URL for connecting apps
DB_URL=postgres://ksc:password@localhost:5432/ksc_data?sslmode=disable

.PHONY: up down sqlc run-back run-front clean

# 🐳 Docker Controls
up:
	docker compose up -d

down:
	docker compose down

# 🧹 Reset the Database (Use with caution!)
# Stops docker, deletes the volume (wiping data), and restarts.
# This forces Postgres to re-run 001_init.sql from scratch.
reset-db:
	docker compose down -v
	docker compose up -d
	@echo "Database has been wiped and re-initialized."

# 🛠️ Code Generation
sqlc:
	cd backend && sqlc generate

# 🚀 Run the Apps
run-back:
	cd backend && go run ./cmd/api

run-front:
	cd frontend && npm run dev

# 📥 Import Excel data
# Usage: make import FILE=path/to/file.xlsx JOB=job123 DATE=2026-01
import:
	cd backend && go run ./cmd/import -file="$(FILE)" -job="$(JOB)" -date="$(DATE)"

# 📥 Import job cost ledger from Excel
# Usage: make import-ledger FILE=path/to/file.xlsx
import-ledger:
	cd backend && go run ./cmd/import-ledger -file="$(FILE)"

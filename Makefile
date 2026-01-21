ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Database URL for connecting apps
DB_URL=postgres://ksc:password@localhost:5432/ksc_data?sslmode=disable

.PHONY: up down sqlc run-back run-front clean deploy migrate migrate-status migrate-down install-goose

# 🐳 Docker Controls
up:
	docker compose up -d

down:
	docker compose down

# 🧹 Reset the Database (Use with caution!)
# Stops docker, deletes the volume (wiping data), restarts, and runs migrations.
reset-db:
	docker compose down -v
	docker compose up -d db
	@echo "Waiting for database to be ready..."
	@sleep 3
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" up
	@echo "Database has been wiped and re-initialized with migrations."

# 🛠️ Code Generation
sqlc:
	cd backend && sqlc generate

# 🗃️ Database Migrations (using goose)
# Install goose first: make install-goose
MIGRATIONS_DIR=backend/sql/schema

install-goose:
	go install github.com/pressly/goose/v3/cmd/goose@latest

migrate:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" up

migrate-status:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" status

migrate-down:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" down

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

# 🚀 Production Deployment
# Pulls latest code, runs migrations, rebuilds and restarts app containers.
# Database container and volume are NOT touched.
deploy:
	@echo "Pulling latest changes..."
	git pull
	@echo "Running database migrations..."
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" up
	@echo "Stopping application containers..."
	docker compose --profile prod down
	@echo "Rebuilding and starting services..."
	docker compose --profile prod up -d --build
	@echo "Deployment complete!"
	@echo "Checking container status..."
	docker compose --profile prod ps

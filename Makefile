ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Database URL for connecting apps
DB_URL=postgres://ksc:password@localhost:5432/ksc_data?sslmode=disable

.PHONY: up down sqlc run-back run-front clean deploy migrate migrate-status migrate-down

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
	docker compose run --rm --build migrate up
	@echo "Database has been wiped and re-initialized with migrations."

# 🛠️ Code Generation
sqlc:
	cd backend && sqlc generate

# 🗃️ Database Migrations (using goose in docker)
migrate:
	docker compose run --rm --build migrate up

migrate-status:
	docker compose run --rm --build migrate status

migrate-down:
	docker compose run --rm --build migrate down

# One-time: Mark existing migrations as applied (for databases created before goose)
migrate-init:
	@echo "Creating goose version table and marking existing migrations as applied..."
	docker exec ksc_db psql -U ksc -d ksc_data -c " \
		CREATE TABLE IF NOT EXISTS goose_db_version ( \
			id SERIAL PRIMARY KEY, \
			version_id BIGINT NOT NULL, \
			is_applied BOOLEAN NOT NULL, \
			tstamp TIMESTAMP DEFAULT now() \
		); \
		INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, true) ON CONFLICT DO NOTHING; \
		INSERT INTO goose_db_version (version_id, is_applied) VALUES (1, true); \
		INSERT INTO goose_db_version (version_id, is_applied) VALUES (2, true); \
		INSERT INTO goose_db_version (version_id, is_applied) VALUES (3, true); \
	"
	@echo "Done. Future migrations will run normally."

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
	@echo "Ensuring database is running..."
	docker compose up -d db
	@echo "Running database migrations..."
	docker compose run --rm --build migrate up
	@echo "Stopping application containers..."
	docker compose --profile prod down
	@echo "Rebuilding and starting services..."
	docker compose --profile prod up -d --build
	@echo "Deployment complete!"
	@echo "Checking container status..."
	docker compose --profile prod ps

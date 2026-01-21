-- +goose Up

-- Add columns for bid import (from Excel cost estimating export)
ALTER TABLE job_items
  ADD COLUMN cost_method TEXT,
  ADD COLUMN production_rate NUMERIC,
  ADD COLUMN production_units TEXT,
  ADD COLUMN man_hours NUMERIC NOT NULL DEFAULT 0,
  ADD COLUMN production_hours NUMERIC NOT NULL DEFAULT 0,
  ADD COLUMN crew_days NUMERIC NOT NULL DEFAULT 0,
  ADD COLUMN plug NUMERIC NOT NULL DEFAULT 0,
  ADD COLUMN labor NUMERIC NOT NULL DEFAULT 0,
  ADD COLUMN equip NUMERIC NOT NULL DEFAULT 0,
  ADD COLUMN misc NUMERIC NOT NULL DEFAULT 0,
  ADD COLUMN material NUMERIC NOT NULL DEFAULT 0,
  ADD COLUMN sub NUMERIC NOT NULL DEFAULT 0,
  ADD COLUMN trucking NUMERIC NOT NULL DEFAULT 0,
  ADD COLUMN indirect NUMERIC NOT NULL DEFAULT 0,
  ADD COLUMN bond NUMERIC NOT NULL DEFAULT 0,
  ADD COLUMN overhead NUMERIC NOT NULL DEFAULT 0,
  ADD COLUMN profit NUMERIC NOT NULL DEFAULT 0;

-- +goose Down

ALTER TABLE job_items
  DROP COLUMN cost_method,
  DROP COLUMN production_rate,
  DROP COLUMN production_units,
  DROP COLUMN man_hours,
  DROP COLUMN production_hours,
  DROP COLUMN crew_days,
  DROP COLUMN plug,
  DROP COLUMN labor,
  DROP COLUMN equip,
  DROP COLUMN misc,
  DROP COLUMN material,
  DROP COLUMN sub,
  DROP COLUMN trucking,
  DROP COLUMN indirect,
  DROP COLUMN bond,
  DROP COLUMN overhead,
  DROP COLUMN profit;

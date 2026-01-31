-- Add columns for bid import (from Excel cost estimating export)
-- Using DO block to handle idempotency
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='job_items' AND column_name='cost_method') THEN
        ALTER TABLE job_items ADD COLUMN cost_method TEXT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='job_items' AND column_name='production_rate') THEN
        ALTER TABLE job_items ADD COLUMN production_rate NUMERIC;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='job_items' AND column_name='production_units') THEN
        ALTER TABLE job_items ADD COLUMN production_units TEXT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='job_items' AND column_name='man_hours') THEN
        ALTER TABLE job_items ADD COLUMN man_hours NUMERIC NOT NULL DEFAULT 0;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='job_items' AND column_name='production_hours') THEN
        ALTER TABLE job_items ADD COLUMN production_hours NUMERIC NOT NULL DEFAULT 0;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='job_items' AND column_name='crew_days') THEN
        ALTER TABLE job_items ADD COLUMN crew_days NUMERIC NOT NULL DEFAULT 0;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='job_items' AND column_name='plug') THEN
        ALTER TABLE job_items ADD COLUMN plug NUMERIC NOT NULL DEFAULT 0;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='job_items' AND column_name='labor') THEN
        ALTER TABLE job_items ADD COLUMN labor NUMERIC NOT NULL DEFAULT 0;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='job_items' AND column_name='equip') THEN
        ALTER TABLE job_items ADD COLUMN equip NUMERIC NOT NULL DEFAULT 0;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='job_items' AND column_name='misc') THEN
        ALTER TABLE job_items ADD COLUMN misc NUMERIC NOT NULL DEFAULT 0;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='job_items' AND column_name='material') THEN
        ALTER TABLE job_items ADD COLUMN material NUMERIC NOT NULL DEFAULT 0;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='job_items' AND column_name='sub') THEN
        ALTER TABLE job_items ADD COLUMN sub NUMERIC NOT NULL DEFAULT 0;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='job_items' AND column_name='trucking') THEN
        ALTER TABLE job_items ADD COLUMN trucking NUMERIC NOT NULL DEFAULT 0;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='job_items' AND column_name='indirect') THEN
        ALTER TABLE job_items ADD COLUMN indirect NUMERIC NOT NULL DEFAULT 0;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='job_items' AND column_name='bond') THEN
        ALTER TABLE job_items ADD COLUMN bond NUMERIC NOT NULL DEFAULT 0;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='job_items' AND column_name='overhead') THEN
        ALTER TABLE job_items ADD COLUMN overhead NUMERIC NOT NULL DEFAULT 0;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='job_items' AND column_name='profit') THEN
        ALTER TABLE job_items ADD COLUMN profit NUMERIC NOT NULL DEFAULT 0;
    END IF;
END $$;

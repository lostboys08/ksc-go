-- +goose Up
-- Job Cost Ledger table for tracking job costs from imported data
CREATE TABLE job_cost_ledger (
  id VARCHAR(64) PRIMARY KEY,
  job VARCHAR(50) NOT NULL,
  phase VARCHAR(50),
  cat VARCHAR(50),
  transaction_type VARCHAR(100),
  transaction_date DATE,
  amount NUMERIC NOT NULL DEFAULT 0,

  created_at TIMESTAMP DEFAULT NOW()
);

-- Index for common lookups
CREATE INDEX idx_job_cost_ledger_job ON job_cost_ledger(job);
CREATE INDEX idx_job_cost_ledger_transaction_date ON job_cost_ledger(transaction_date);

-- +goose Down
DROP INDEX IF EXISTS idx_job_cost_ledger_transaction_date;
DROP INDEX IF EXISTS idx_job_cost_ledger_job;
DROP TABLE IF EXISTS job_cost_ledger;

-- 1. Project Metadata
CREATE TABLE IF NOT EXISTS jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  job_number VARCHAR(50) NOT NULL,
  job_name TEXT NOT NULL,
  contract_value NUMERIC,
  address TEXT,
  scr_number VARCHAR(50),
  contract_complete_date DATE,
  start_date DATE,
  end_date DATE,
  
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(job_number)
);

-- 2. Bid Detail
CREATE TABLE IF NOT EXISTS job_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  job_id UUID NOT NULL REFERENCES jobs(id),
  parent_id UUID REFERENCES job_items(id),
  sort_order INT NOT NULL,
  item_number VARCHAR(50) NOT NULL,
  description TEXT NOT NULL,
  scheduled_value NUMERIC NOT NULL DEFAULT 0,
  job_cost_id VARCHAR(20),
  budget NUMERIC NOT NULL DEFAULT 0,
  qty NUMERIC NOT NULL DEFAULT 0,
  unit TEXT,
  unit_price NUMERIC NOT NULL DEFAULT 0,

  updated_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(job_id, item_number)
);

-- 3. Time-Series Data
CREATE TABLE IF NOT EXISTS pay_applications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  job_item_id UUID NOT NULL REFERENCES job_items(id),
  pay_app_month DATE NOT NULL,
  qty NUMERIC NOT NULL DEFAULT 0,
  stored_materials NUMERIC NOT NULL DEFAULT 0,

  updated_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(job_item_id, pay_app_month)
);

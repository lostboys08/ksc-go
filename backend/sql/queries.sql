-- name: UpsertJob :one
INSERT INTO jobs (
    job_number, job_name, contract_value, address, 
    scr_number, contract_complete_date, start_date, end_date
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (job_number) 
DO UPDATE SET 
    job_name = EXCLUDED.job_name,
    contract_value = EXCLUDED.contract_value,
    address = EXCLUDED.address,
    scr_number = EXCLUDED.scr_number,
    contract_complete_date = EXCLUDED.contract_complete_date,
    start_date = EXCLUDED.start_date,
    end_date = EXCLUDED.end_date,
    updated_at = NOW()
RETURNING id;

-- name: GetAllJobs :many
SELECT id, job_number, job_name FROM jobs ORDER BY job_number;

-- name: GetJobByNumber :one
SELECT id, job_number, job_name FROM jobs WHERE job_number = $1;

-- name: UpsertJobItem :one
INSERT INTO job_items (
    job_id, parent_id, sort_order, item_number, description, 
    scheduled_value, job_cost_id, budget, qty, unit, unit_price
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
ON CONFLICT (job_id, item_number)
DO UPDATE SET 
    parent_id = EXCLUDED.parent_id,
    sort_order = EXCLUDED.sort_order,
    description = EXCLUDED.description,
    scheduled_value = EXCLUDED.scheduled_value,
    job_cost_id = EXCLUDED.job_cost_id,
    budget = EXCLUDED.budget,
    qty = EXCLUDED.qty,
    unit = EXCLUDED.unit,
    unit_price = EXCLUDED.unit_price,
    updated_at = NOW()
RETURNING id;

-- name: DeleteJobItemsByJob :exec
DELETE FROM job_items WHERE job_id = $1;

-- name: UpsertPayApplication :exec
INSERT INTO pay_applications (
    job_item_id, pay_app_month, qty, stored_materials
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (job_item_id, pay_app_month)
DO UPDATE SET
    qty = EXCLUDED.qty,
    stored_materials = EXCLUDED.stored_materials,
    updated_at = NOW();

-- name: InsertPayApplicationIfNotExists :exec
-- Inserts pay application data only if no record exists for this item/month
INSERT INTO pay_applications (
    job_item_id, pay_app_month, qty, stored_materials
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (job_item_id, pay_app_month) DO NOTHING;

-- name: UpdateStoredMaterials :exec
-- Updates only the stored_materials field for an existing pay application
UPDATE pay_applications
SET stored_materials = $3, updated_at = NOW()
WHERE job_item_id = $1 AND pay_app_month = $2;

-- name: GetJobTree :many
WITH RECURSIVE job_tree AS (
    -- 1. Anchor: Select Roots (Items with no parent)
    -- We alias the table as 'root' to prevent ambiguity errors
    SELECT 
        root.id, 
        root.job_id, 
        root.parent_id, 
        root.sort_order, 
        root.item_number, 
        root.description,
        root.scheduled_value, 
        root.budget, 
        root.job_cost_id, 
        root.qty, 
        root.unit, 
        root.unit_price,
        1 AS depth,
        ARRAY[root.sort_order] AS path_order
    FROM job_items root
    WHERE root.job_id = $1 AND root.parent_id IS NULL

    UNION ALL

    -- 2. Recursion: Join Children to Parents
    SELECT 
        c.id, 
        c.job_id, 
        c.parent_id, 
        c.sort_order, 
        c.item_number, 
        c.description,
        c.scheduled_value, 
        c.budget, 
        c.job_cost_id, 
        c.qty, 
        c.unit, 
        c.unit_price,
        p.depth + 1,
        p.path_order || c.sort_order
    FROM job_items c
    JOIN job_tree p ON c.parent_id = p.id
)
-- 3. Sort by the array path to recreate Excel structure
SELECT * FROM job_tree
ORDER BY path_order;

-- name: GetPayAppCumulative :many
-- Fetches cumulative pay application data for a job and specific month
SELECT
    job_item_id,
    pay_app_month,
    job_id,
    parent_id,
    this_month_qty,
    stored_materials,
    total_qty,
    unit_price,
    budget,
    cumulative_qty,
    previous_cumulative_qty,
    remaining_qty,
    percent_complete,
    this_month_amount,
    cumulative_amount,
    previous_cumulative_amount
FROM pay_application_cumulative
WHERE job_id = $1 AND pay_app_month = $2;

-- name: GetDirectChildren :many
-- Fetches direct children of a parent item (for validation)
SELECT
    id,
    parent_id,
    item_number,
    description,
    budget,
    qty,
    unit_price,
    scheduled_value
FROM job_items
WHERE parent_id = $1;

-- name: GetParentItems :many
-- Fetches all parent items (items that have children) for a job
SELECT DISTINCT ji.id, ji.item_number, ji.description, ji.budget, ji.qty, ji.unit_price
FROM job_items ji
WHERE ji.job_id = $1
  AND EXISTS (SELECT 1 FROM job_items child WHERE child.parent_id = ji.id);

-- name: GetChildrenWithPayApps :many
-- Fetches children of a parent along with their pay_app data for a specific month
SELECT
    ji.id,
    ji.item_number,
    ji.qty AS total_qty,
    ji.unit_price,
    COALESCE(pa.qty, '0')::TEXT AS pay_app_qty,
    COALESCE(pac.cumulative_qty, '0')::TEXT AS cumulative_qty,
    COALESCE(pac.previous_cumulative_qty, '0')::TEXT AS previous_cumulative_qty
FROM job_items ji
LEFT JOIN pay_applications pa ON ji.id = pa.job_item_id AND pa.pay_app_month = $2
LEFT JOIN pay_application_cumulative pac ON ji.id = pac.job_item_id AND pac.pay_app_month = $2
WHERE ji.parent_id = $1;

-- name: GetPayAppMonthsForJob :many
-- Gets all distinct months with pay applications for a job
SELECT DISTINCT pa.pay_app_month
FROM pay_applications pa
JOIN job_items ji ON pa.job_item_id = ji.id
WHERE ji.job_id = $1
ORDER BY pa.pay_app_month;

-- name: GetParentPayAppForMonth :one
-- Gets a parent's pay application data for a specific month
SELECT
    pa.qty,
    pac.cumulative_qty::TEXT AS cumulative_qty,
    pac.previous_cumulative_qty::TEXT AS previous_cumulative_qty,
    ji.qty AS total_qty,
    ji.unit_price
FROM pay_applications pa
JOIN job_items ji ON pa.job_item_id = ji.id
LEFT JOIN pay_application_cumulative pac ON pa.job_item_id = pac.job_item_id AND pa.pay_app_month = pac.pay_app_month
WHERE pa.job_item_id = $1 AND pa.pay_app_month = $2;

-- name: InsertJobCostLedger :exec
-- Inserts a job cost ledger entry, skips if hash already exists
INSERT INTO job_cost_ledger (
    id, job, phase, cat, transaction_type, transaction_date, amount
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
ON CONFLICT (id) DO NOTHING;

-- name: GetJobCostLedgerByJob :many
-- Fetches all job cost ledger entries for a specific job
SELECT id, job, phase, cat, transaction_type, transaction_date, amount, created_at
FROM job_cost_ledger
WHERE job = $1
ORDER BY transaction_date;

-- name: GetMonthlyPerformance :many
-- Fetches monthly cost and pay application totals for a job
-- Only includes months that have pay applications with non-zero totals
WITH monthly_costs AS (
    SELECT
        DATE_TRUNC('month', transaction_date)::DATE AS month,
        SUM(amount) AS cost_total
    FROM job_cost_ledger
    WHERE job = $1
      AND transaction_type IN ('AP cost', 'JC cost', 'PR cost')
      AND transaction_date IS NOT NULL
    GROUP BY DATE_TRUNC('month', transaction_date)
),
monthly_pay_apps AS (
    SELECT
        pa.pay_app_month AS month,
        SUM(pa.qty * ji.unit_price) AS pay_app_total
    FROM pay_applications pa
    JOIN job_items ji ON pa.job_item_id = ji.id
    JOIN jobs j ON ji.job_id = j.id
    WHERE j.job_number = $1
    GROUP BY pa.pay_app_month
    HAVING SUM(pa.qty * ji.unit_price) > 0
),
cumulative_costs AS (
    SELECT
        month,
        cost_total,
        SUM(cost_total) OVER (ORDER BY month) AS running_cost
    FROM monthly_costs
)
SELECT
    mpa.month,
    CAST(COALESCE(cc.running_cost, 0) AS BIGINT) AS cost_total,
    CAST(mpa.pay_app_total AS BIGINT) AS pay_app_total,
    CAST(COALESCE(cc.running_cost, 0) AS BIGINT) AS cumulative_cost,
    CAST(SUM(mpa.pay_app_total) OVER (ORDER BY mpa.month) AS BIGINT) AS cumulative_pay_app
FROM monthly_pay_apps mpa
LEFT JOIN LATERAL (
    SELECT SUM(cost_total) AS running_cost
    FROM monthly_costs mc
    WHERE mc.month <= mpa.month
) cc ON true
ORDER BY mpa.month;

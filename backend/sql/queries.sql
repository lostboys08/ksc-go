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
-- Fetches monthly cost and billed totals for a job from job_cost_ledger
-- Costs = AP cost, JC cost, PR cost; Billed = work billed
WITH monthly_costs AS (
    SELECT
        DATE_TRUNC('month', jcl.transaction_date)::DATE AS month,
        SUM(jcl.amount) AS cost_total
    FROM job_cost_ledger jcl
    WHERE jcl.job = $1
      AND jcl.transaction_type IN ('AP cost', 'JC cost', 'PR cost')
      AND jcl.transaction_date IS NOT NULL
    GROUP BY DATE_TRUNC('month', jcl.transaction_date)
),
monthly_billed AS (
    SELECT
        DATE_TRUNC('month', jcl.transaction_date)::DATE AS month,
        SUM(jcl.amount) AS billed_total
    FROM job_cost_ledger jcl
    WHERE jcl.job = $1
      AND jcl.transaction_type = 'work billed'
      AND jcl.transaction_date IS NOT NULL
    GROUP BY DATE_TRUNC('month', jcl.transaction_date)
    HAVING SUM(jcl.amount) > 0
)
SELECT
    mb.month,
    COALESCE((
        SELECT SUM(mc.cost_total)
        FROM monthly_costs mc
        WHERE mc.month <= mb.month
    ), 0)::BIGINT AS cost_total,
    mb.billed_total::BIGINT AS pay_app_total,
    COALESCE((
        SELECT SUM(mc.cost_total)
        FROM monthly_costs mc
        WHERE mc.month <= mb.month
    ), 0)::BIGINT AS cumulative_cost,
    SUM(mb.billed_total) OVER (ORDER BY mb.month)::BIGINT AS cumulative_pay_app
FROM monthly_billed mb
ORDER BY mb.month;

-- name: GetCostPerformanceIndex :many
-- Fetches monthly CPI data: Earned Value / Actual Costs
-- Earned Value = (cumulative qty / total qty) * original budget
-- Falls back to sum of scheduled_value if contract_value is not set
WITH job_info AS (
    SELECT id, job_number, contract_value
    FROM jobs
    WHERE job_number = $1
),
total_scheduled AS (
    -- Sum of all scheduled quantities and calculated values (only leaf items with qty > 0)
    SELECT
        SUM(ji.qty) AS total_qty,
        SUM(ji.qty * ji.unit_price) AS total_contract_value
    FROM job_items ji
    JOIN job_info j ON ji.job_id = j.id
    WHERE ji.qty > 0
),
budget_value AS (
    -- Use contract_value if set, otherwise fall back to sum of qty * unit_price
    SELECT COALESCE(j.contract_value, ts.total_contract_value, 0) AS budget
    FROM job_info j
    CROSS JOIN total_scheduled ts
),
monthly_cumulative_qty AS (
    -- Cumulative quantity billed per month
    SELECT
        pa.pay_app_month AS month,
        SUM(pa.qty) AS month_qty,
        SUM(SUM(pa.qty)) OVER (ORDER BY pa.pay_app_month) AS cumulative_qty
    FROM pay_applications pa
    JOIN job_items ji ON pa.job_item_id = ji.id
    JOIN job_info j ON ji.job_id = j.id
    GROUP BY pa.pay_app_month
),
monthly_costs AS (
    SELECT
        DATE_TRUNC('month', transaction_date)::DATE AS month,
        SUM(amount) AS cost_total
    FROM job_cost_ledger
    WHERE job = $1
      AND transaction_type IN ('AP cost', 'JC cost', 'PR cost')
      AND transaction_date IS NOT NULL
    GROUP BY DATE_TRUNC('month', transaction_date)
)
SELECT
    mcq.month,
    COALESCE(bv.budget, 0)::BIGINT AS budget,
    COALESCE(ts.total_qty, 0)::TEXT AS total_scheduled_qty,
    mcq.cumulative_qty::TEXT AS cumulative_qty,
    CASE
        WHEN COALESCE(ts.total_qty, 0) > 0
        THEN ROUND((mcq.cumulative_qty / ts.total_qty) * 100, 2)::TEXT
        ELSE '0'
    END AS percent_complete,
    CASE
        WHEN COALESCE(ts.total_qty, 0) > 0 AND COALESCE(bv.budget, 0) > 0
        THEN ROUND((mcq.cumulative_qty / ts.total_qty) * bv.budget, 2)::BIGINT
        ELSE 0::BIGINT
    END AS earned_value,
    COALESCE(SUM(mc.cost_total) OVER (ORDER BY mcq.month), 0)::BIGINT AS actual_cost,
    CASE
        WHEN COALESCE(SUM(mc.cost_total) OVER (ORDER BY mcq.month), 0) > 0
             AND COALESCE(ts.total_qty, 0) > 0
             AND COALESCE(bv.budget, 0) > 0
        THEN ROUND(
            ((mcq.cumulative_qty / ts.total_qty) * bv.budget) /
            SUM(mc.cost_total) OVER (ORDER BY mcq.month),
            2
        )::TEXT
        ELSE '0'
    END AS cpi
FROM monthly_cumulative_qty mcq
CROSS JOIN budget_value bv
CROSS JOIN total_scheduled ts
LEFT JOIN monthly_costs mc ON mc.month = mcq.month
ORDER BY mcq.month;

-- name: GetOverBudgetPhases :many
-- Fetches phase codes where actual costs exceed budget
-- Uses "Original estimate" from job_cost_ledger as the budget source
WITH phase_costs AS (
    SELECT
        jcl.phase,
        SUM(jcl.amount) AS actual_cost
    FROM job_cost_ledger jcl
    WHERE jcl.job = $1
      AND jcl.transaction_type IN ('AP cost', 'JC cost', 'PR cost')
    GROUP BY jcl.phase
),
phase_budgets AS (
    SELECT
        jcl.phase,
        SUM(jcl.amount) AS budget
    FROM job_cost_ledger jcl
    WHERE jcl.job = $1
      AND jcl.transaction_type = 'Original estimate'
    GROUP BY jcl.phase
),
phase_descriptions AS (
    SELECT
        ji.job_cost_id AS phase,
        STRING_AGG(DISTINCT ji.description, ', ') AS descriptions
    FROM job_items ji
    WHERE ji.job_id = (SELECT j.id FROM jobs j WHERE j.job_number = $1)
      AND ji.job_cost_id IS NOT NULL
    GROUP BY ji.job_cost_id
)
SELECT
    COALESCE(pb.phase, pc.phase) AS phase,
    COALESCE(pd.descriptions, '') AS description,
    COALESCE(pb.budget, 0)::BIGINT AS budget,
    COALESCE(pc.actual_cost, 0)::BIGINT AS actual_cost,
    (COALESCE(pc.actual_cost, 0) - COALESCE(pb.budget, 0))::BIGINT AS variance
FROM phase_budgets pb
FULL OUTER JOIN phase_costs pc ON pb.phase = pc.phase
LEFT JOIN phase_descriptions pd ON COALESCE(pb.phase, pc.phase) = pd.phase
WHERE COALESCE(pc.actual_cost, 0) > COALESCE(pb.budget, 0)
ORDER BY (COALESCE(pc.actual_cost, 0) - COALESCE(pb.budget, 0)) DESC;

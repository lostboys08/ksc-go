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

-- Cumulative view for pay application calculations
CREATE OR REPLACE VIEW pay_application_cumulative AS
WITH ordered_months AS (
    SELECT
        pa.job_item_id,
        pa.pay_app_month,
        pa.qty,
        pa.stored_materials,
        ji.qty AS total_qty,
        ji.unit_price,
        ji.budget,
        ji.parent_id,
        ji.job_id
    FROM pay_applications pa
    JOIN job_items ji ON pa.job_item_id = ji.id
),
cumulative AS (
    SELECT
        job_item_id,
        pay_app_month,
        qty AS this_month_qty,
        stored_materials,
        total_qty,
        unit_price,
        budget,
        parent_id,
        job_id,
        SUM(qty::NUMERIC) OVER (
            PARTITION BY job_item_id
            ORDER BY pay_app_month
            ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
        ) AS cumulative_qty_num,
        COALESCE(
            SUM(qty::NUMERIC) OVER (
                PARTITION BY job_item_id
                ORDER BY pay_app_month
                ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING
            ),
            0
        ) AS previous_cumulative_qty_num
    FROM ordered_months
)
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
    cumulative_qty_num::TEXT AS cumulative_qty,
    previous_cumulative_qty_num::TEXT AS previous_cumulative_qty,
    (total_qty::NUMERIC - cumulative_qty_num)::TEXT AS remaining_qty,
    CASE
        WHEN total_qty::NUMERIC = 0 THEN '0'
        ELSE ROUND((cumulative_qty_num / total_qty::NUMERIC) * 100, 4)::TEXT
    END AS percent_complete,
    (this_month_qty::NUMERIC * unit_price::NUMERIC)::TEXT AS this_month_amount,
    (cumulative_qty_num * unit_price::NUMERIC)::TEXT AS cumulative_amount,
    (previous_cumulative_qty_num * unit_price::NUMERIC)::TEXT AS previous_cumulative_amount
FROM cumulative;

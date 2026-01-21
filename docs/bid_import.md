# Excel Bid Upload Parser

## Overview

This document describes how to parse a bid export Excel file and import it into the `job_items` table, preserving the hierarchical cost structure.

---

## Step 1: File Format

### Column Headers

| Excel Column | DB Column | Notes |
|--------------|-----------|-------|
| Item # | `item_number` | Only populated for pay items (top-level bid items) |
| Total Direct Cost | `budget` | Used to determine parent-child relationships |
| Job Cost ID | `job_cost_id` | |
| Description | `description` | |
| Quantity | `qty` | |
| UM | `unit` | Unit of measure |
| Cost Method | `cost_method` | Key field for hierarchy logic (see below) |
| Selected Sub. | — | Do not map |
| Total Bid Price | `scheduled_value` | |
| Production Rate | `production_rate` | Non-blank indicates a Crew item |
| Production Method | `production_units` | |
| Total Man Hours | `man_hours` | |
| Total Prod. Hours | `production_hours` | |
| Crew Days | `crew_days` | |
| Total Plug | `plug` | |
| Total Labor | `labor` | |
| Total Equip. | `equip` | |
| Total Misc. | `misc` | |
| Total Material | `material` | |
| Total Subcontracted | `sub` | |
| Total Trucking | `trucking` | |
| Total Ind. Cost | `indirect` | |
| Total Bond | `bond` | |
| Total OH | `overhead` | |
| Total Profit | `profit` | |
| Selected Vendor | — | Do not map |

### Additional DB Columns (not from Excel)

| DB Column | Description |
|-----------|-------------|
| `id` | UUID, auto-generated |
| `job_id` | FK to parent job |
| `parent_id` | FK to parent job_item (for hierarchy) |
| `sort_order` | Preserves original row order |
| `updated_at` | Timestamp |
| `unit_price` | Calculated: `scheduled_value / qty` |

---

## Step 2: Understanding the Hierarchy

The Excel file represents a tree structure flattened into rows. The hierarchy has these levels:

```
Pay Item (Item # is populated)
└── Detail (Cost Method = "Detail")
    ├── Crew (Cost Method is blank, Production Rate is populated)
    │   ├── Labor
    │   ├── Equipment
    │   └── ...
    ├── Material
    ├── Labor
    ├── Equipment
    ├── Trucking
    └── ...
```

### Item Type Identification

| Condition | Item Type | `cost_method` in DB | Can Have Children |
|-----------|-----------|---------------------|-------------------|
| `Item #` has a value | Pay Item | `"Pay Item"` | Yes |
| `Cost Method` = "Detail" | Detail | `"Detail"` | Yes |
| `Cost Method` = "Subcontracted" | Subcontracted | `"Subcontracted"` | **No** |
| `Cost Method` is blank AND `Production Rate` has a value | Crew | `"Crew"` | Yes |
| `Cost Method` is blank AND `Production Rate` is blank | Cost Component | Inherit from context | No |
| `Cost Method` has any other value | Cost Component | Use the value directly | No |

---

## Step 3: Parsing Algorithm (Single Pass)

Process rows sequentially using a stack where each entry tracks its own running sum. The key insight: **pop completed parents *before* assigning the current row's parent**.

### Stack Entry Structure

```
StackEntry {
  item: JobItem       // The item itself (so we can reference its ID)
  target: float       // budget - what children should sum to
  sum: float          // running total of children's budgets
}
```

### Algorithm

```
TOLERANCE = 0.01  // for floating-point comparison
stack = []

for each row in excel_file:
    // Skip empty rows
    if row.Description is blank:
        continue

    // Create the job_item record
    item = new JobItem from row
    item.sort_order = row_index

    // Determine item type and set cost_method
    if row.ItemNumber is not blank:
        item.cost_method = "Pay Item"
        can_have_children = true
    else if row.CostMethod == "Detail":
        item.cost_method = "Detail"
        can_have_children = true
    else if row.CostMethod is blank AND row.ProductionRate is not blank:
        item.cost_method = "Crew"
        can_have_children = true
    else:
        // Cost component - use CostMethod value or infer from context
        item.cost_method = row.CostMethod or "Cost"
        can_have_children = false

    // STEP 1: Pop any completed parents BEFORE assigning this row
    while stack is not empty AND stack.top.sum >= (stack.top.target - TOLERANCE):
        stack.pop()

    // STEP 2: Assign parent
    if stack is empty:
        item.parent_id = null
    else:
        item.parent_id = stack.top.item.id
        // Add this item's budget to parent's running sum
        stack.top.sum += row.TotalDirectCost

    // STEP 3: If this item can have children, push it onto the stack
    if can_have_children AND row.TotalDirectCost > 0:
        stack.push({
            item: item,
            target: row.TotalDirectCost,
            sum: 0
        })

    // Save the item
    save(item)

// After processing all rows, calculate unit_price
for each item where qty > 0:
    item.unit_price = item.scheduled_value / item.qty
```

### Walkthrough Example

Given this nested structure:
```
Pay Item ($1000)
├── Detail A ($600)
│   ├── Labor ($300)
│   └── Equipment ($300)
└── Detail B ($400)
    └── Material ($400)
```

| Row | Action | Stack After |
|-----|--------|-------------|
| Pay Item ($1000) | Push | `[{PayItem, target=1000, sum=0}]` |
| Detail A ($600) | PayItem.sum=0 < 1000, assign to PayItem, add 600, push | `[{PayItem, 1000, 600}, {DetailA, 600, 0}]` |
| Labor ($300) | DetailA.sum=0 < 600, assign to DetailA, add 300 | `[{PayItem, 1000, 600}, {DetailA, 600, 300}]` |
| Equipment ($300) | DetailA.sum=300 < 600, assign to DetailA, add 300 | `[{PayItem, 1000, 600}, {DetailA, 600, 600}]` |
| Detail B ($400) | **DetailA.sum=600 ≥ 600, POP!** Then PayItem.sum=600 < 1000, assign to PayItem, add 400, push | `[{PayItem, 1000, 1000}, {DetailB, 400, 0}]` |
| Material ($400) | DetailB.sum=0 < 400, assign to DetailB, add 400 | `[{PayItem, 1000, 1000}, {DetailB, 400, 400}]` |
| (end of file) | Pop remaining completed items | `[]` |

### Handling Deeply Nested Details

The same logic works for arbitrary nesting depth:

```
Pay Item ($1000)
└── Detail A ($1000)
    ├── Detail B ($600)
    │   └── Labor ($600)
    └── Detail C ($400)
        └── Material ($400)
```

| Row | Stack After |
|-----|-------------|
| Pay Item ($1000) | `[{PayItem, 1000, 0}]` |
| Detail A ($1000) | `[{PayItem, 1000, 1000}, {DetailA, 1000, 0}]` |
| Detail B ($600) | `[{PayItem, 1000, 1000}, {DetailA, 1000, 600}, {DetailB, 600, 0}]` |
| Labor ($600) | `[{PayItem, 1000, 1000}, {DetailA, 1000, 600}, {DetailB, 600, 600}]` |
| Detail C ($400) | Pop DetailB (full), assign to DetailA → `[{PayItem, 1000, 1000}, {DetailA, 1000, 1000}, {DetailC, 400, 0}]` |
| Material ($400) | `[{PayItem, 1000, 1000}, {DetailA, 1000, 1000}, {DetailC, 400, 400}]` |

---

## Step 4: Edge Cases

1. **Rounding errors**: Use `TOLERANCE = 0.01` when comparing sums (already in algorithm)
2. **Empty rows**: Skip rows where Description is blank
3. **Zero budget items**: Don't push zero-budget parents to stack (they'd immediately be "complete"). Assign them a parent but don't track their children via sum.
4. **Nested crews**: Handled automatically - Crews are pushed to stack like Details
5. **Multiple Details under one Pay Item**: Handled automatically - each Detail's budget is added to Pay Item's running sum
6. **Budget mismatch**: If children don't sum exactly to parent (data error), the algorithm still works but parent may pop early or late. Log a warning if `abs(sum - target) > TOLERANCE` when popping.
7. **Sibling detection**: Two items are siblings if they share the same parent. The algorithm handles this naturally - when Detail A completes and pops, Detail B gets assigned to the same Pay Item parent.

---

## Step 5: Implementation Notes

1. Parse the Excel file using a library that preserves column order
2. Assign `sort_order` based on original row index to preserve display order
3. Use the recursive `GetJobTree` query to retrieve the hierarchy for display

### Batching Strategy

**Do not insert rows one-by-one.** Instead, use this approach:

1. **Generate UUIDs client-side** - Before processing, generate all UUIDs in Go using `uuid.New()`. This allows us to know each item's ID before it's inserted, so we can set `parent_id` references in memory.

2. **Build all records in memory first** - Run the full parsing algorithm, creating all `JobItem` structs with their `id` and `parent_id` fields populated.

3. **Batch insert** - Insert all records in a single database operation. Options:
   - **Single multi-row INSERT**: Build one statement with all values
   - **COPY protocol**: Use `pgx.CopyFrom()` for best performance with large files
   - **Prepared statement in transaction**: Loop with a prepared statement inside one transaction

#### Example using pgx CopyFrom (recommended for large imports)

```go
// After parsing, items is []JobItem with IDs and parent_ids already set
columns := []string{"id", "job_id", "parent_id", "item_number", "description", ...}

_, err := conn.CopyFrom(
    ctx,
    pgx.Identifier{"job_items"},
    columns,
    pgx.CopyFromSlice(len(items), func(i int) ([]any, error) {
        item := items[i]
        return []any{
            item.ID,
            item.JobID,
            item.ParentID,  // already references another item's pre-generated UUID
            item.ItemNumber,
            item.Description,
            // ... rest of fields
        }, nil
    }),
)
```

#### Example using multi-row INSERT

```go
// Build a single INSERT with multiple VALUE clauses
// INSERT INTO job_items (id, job_id, parent_id, ...) VALUES
//   ($1, $2, $3, ...),
//   ($4, $5, $6, ...),
//   ...

valueStrings := make([]string, 0, len(items))
valueArgs := make([]any, 0, len(items)*numColumns)

for i, item := range items {
    offset := i * numColumns
    valueStrings = append(valueStrings, fmt.Sprintf(
        "($%d, $%d, $%d, ...)",
        offset+1, offset+2, offset+3, // ...
    ))
    valueArgs = append(valueArgs, item.ID, item.JobID, item.ParentID, ...)
}

query := fmt.Sprintf(
    "INSERT INTO job_items (id, job_id, parent_id, ...) VALUES %s",
    strings.Join(valueStrings, ","),
)
_, err := db.Exec(ctx, query, valueArgs...)
```

#### Key Point

The reason this works: **UUIDs are generated before any database interaction**. When we push an item to the stack, we already know its `id`. When a child references `stack.top.item.id` as its `parent_id`, that UUID is already determined. No database round-trip needed until the final batch insert.


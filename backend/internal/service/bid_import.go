package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lostboys08/ksc-go/backend/internal/database"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

// bidColumnMap holds the column indices for the bid import Excel file.
type bidColumnMap struct {
	ItemNumber       int // Item #
	TotalDirectCost  int // Total Direct Cost (budget)
	JobCostID        int // Job Cost ID
	Description      int // Description
	Quantity         int // Quantity
	UM               int // Unit of Measure
	CostMethod       int // Cost Method
	TotalBidPrice    int // Total Bid Price (scheduled_value)
	ProductionRate   int // Production Rate
	ProductionMethod int // Production Method (production_units)
	TotalManHours    int // Total Man Hours
	TotalProdHours   int // Total Prod. Hours
	CrewDays         int // Crew Days
	TotalPlug        int // Total Plug
	TotalLabor       int // Total Labor
	TotalEquip       int // Total Equip.
	TotalMisc        int // Total Misc.
	TotalMaterial    int // Total Material
	TotalSubcontract int // Total Subcontracted
	TotalTrucking    int // Total Trucking
	TotalIndCost     int // Total Ind. Cost
	TotalBond        int // Total Bond
	TotalOH          int // Total OH (overhead)
	TotalProfit      int // Total Profit
}

// bidHeaderPatterns maps field names to possible header variations.
var bidHeaderPatterns = map[string][]string{
	"itemNumber":       {"item #", "item", "item number"},
	"totalDirectCost":  {"total direct cost", "direct cost"},
	"jobCostID":        {"job cost id", "job cost", "cost id", "phase"},
	"description":      {"description", "desc"},
	"quantity":         {"quantity", "qty"},
	"um":               {"um", "uom", "unit"},
	"costMethod":       {"cost method"},
	"totalBidPrice":    {"total bid price", "bid price", "scheduled value"},
	"productionRate":   {"production rate", "prod rate"},
	"productionMethod": {"production method", "prod method"},
	"totalManHours":    {"total man hours", "man hours"},
	"totalProdHours":   {"total prod. hours", "total prod hours", "prod hours"},
	"crewDays":         {"crew days"},
	"totalPlug":        {"total plug", "plug"},
	"totalLabor":       {"total labor", "labor"},
	"totalEquip":       {"total equip", "total equip.", "equip", "equipment"},
	"totalMisc":        {"total misc", "total misc.", "misc"},
	"totalMaterial":    {"total material", "material"},
	"totalSubcontract": {"total subcontracted", "subcontracted", "sub"},
	"totalTrucking":    {"total trucking", "trucking"},
	"totalIndCost":     {"total ind. cost", "total ind cost", "indirect cost", "indirect"},
	"totalBond":        {"total bond", "bond"},
	"totalOH":          {"total oh", "overhead", "oh"},
	"totalProfit":      {"total profit", "profit"},
}

// stackEntry tracks parent context during hierarchy parsing.
type stackEntry struct {
	item   *database.InsertBidItemParams
	target decimal.Decimal // budget - what children should sum to
	sum    decimal.Decimal // running total of children's budgets
}

const budgetTolerance = 0.01

// ImportBid imports a bid export Excel file for a specific job.
func ImportBid(ctx context.Context, f *excelize.File, q *database.Queries, jobNumber, jobName string) (*UploadResult, error) {
	// Get or create job
	jobID, err := getOrCreateJob(ctx, q, jobNumber, jobName)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create job: %w", err)
	}

	// Delete existing job items for this job (re-import)
	err = q.DeleteJobItemsByJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to clear existing job items: %w", err)
	}

	// Parse and import
	rowsProcessed, err := parseBidFile(ctx, f, q, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bid file: %w", err)
	}

	return &UploadResult{
		Success:       true,
		Message:       fmt.Sprintf("Successfully imported bid for job %s (%d items)", jobNumber, rowsProcessed),
		RowsProcessed: rowsProcessed,
	}, nil
}

// parseBidFile parses the bid Excel file and inserts items with hierarchy.
func parseBidFile(ctx context.Context, f *excelize.File, q *database.Queries, jobID uuid.UUID) (int, error) {
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return 0, fmt.Errorf("Excel file has no sheets")
	}

	sheetName := sheets[0]
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return 0, fmt.Errorf("failed to read sheet: %w", err)
	}

	if len(rows) < 2 {
		return 0, fmt.Errorf("sheet has no data rows")
	}

	// Find header row and build column map
	headerRow, colMap, err := findBidHeaders(rows)
	if err != nil {
		return 0, fmt.Errorf("failed to find headers: %w", err)
	}

	// Build all items in memory first
	items, err := buildBidItems(rows, headerRow, colMap, jobID)
	if err != nil {
		return 0, fmt.Errorf("failed to build items: %w", err)
	}

	// Insert all items in batch
	for _, item := range items {
		err := q.InsertBidItem(ctx, item)
		if err != nil {
			return 0, fmt.Errorf("failed to insert item %s: %w", item.Description, err)
		}
	}

	return len(items), nil
}

// findBidHeaders scans for the header row and builds the column map.
func findBidHeaders(rows [][]string) (int, *bidColumnMap, error) {
	colMap := &bidColumnMap{
		ItemNumber:       -1,
		TotalDirectCost:  -1,
		JobCostID:        -1,
		Description:      -1,
		Quantity:         -1,
		UM:               -1,
		CostMethod:       -1,
		TotalBidPrice:    -1,
		ProductionRate:   -1,
		ProductionMethod: -1,
		TotalManHours:    -1,
		TotalProdHours:   -1,
		CrewDays:         -1,
		TotalPlug:        -1,
		TotalLabor:       -1,
		TotalEquip:       -1,
		TotalMisc:        -1,
		TotalMaterial:    -1,
		TotalSubcontract: -1,
		TotalTrucking:    -1,
		TotalIndCost:     -1,
		TotalBond:        -1,
		TotalOH:          -1,
		TotalProfit:      -1,
	}

	// Scan first 10 rows for headers
	maxScan := 10
	if len(rows) < maxScan {
		maxScan = len(rows)
	}

	for rowIdx := 0; rowIdx < maxScan; rowIdx++ {
		row := rows[rowIdx]
		matchCount := 0

		for colIdx, cell := range row {
			cellLower := strings.ToLower(strings.TrimSpace(cell))
			if cellLower == "" {
				continue
			}

			// Check each pattern
			for field, patterns := range bidHeaderPatterns {
				for _, pattern := range patterns {
					if cellLower == pattern || strings.Contains(cellLower, pattern) {
						setColumnIndex(colMap, field, colIdx)
						matchCount++
						break
					}
				}
			}
		}

		// If we matched enough headers, this is likely the header row
		if matchCount >= 5 {
			return rowIdx, colMap, nil
		}
	}

	return 0, nil, fmt.Errorf("could not find header row with bid columns")
}

// setColumnIndex sets the appropriate field in bidColumnMap.
func setColumnIndex(colMap *bidColumnMap, field string, idx int) {
	switch field {
	case "itemNumber":
		colMap.ItemNumber = idx
	case "totalDirectCost":
		colMap.TotalDirectCost = idx
	case "jobCostID":
		colMap.JobCostID = idx
	case "description":
		colMap.Description = idx
	case "quantity":
		colMap.Quantity = idx
	case "um":
		colMap.UM = idx
	case "costMethod":
		colMap.CostMethod = idx
	case "totalBidPrice":
		colMap.TotalBidPrice = idx
	case "productionRate":
		colMap.ProductionRate = idx
	case "productionMethod":
		colMap.ProductionMethod = idx
	case "totalManHours":
		colMap.TotalManHours = idx
	case "totalProdHours":
		colMap.TotalProdHours = idx
	case "crewDays":
		colMap.CrewDays = idx
	case "totalPlug":
		colMap.TotalPlug = idx
	case "totalLabor":
		colMap.TotalLabor = idx
	case "totalEquip":
		colMap.TotalEquip = idx
	case "totalMisc":
		colMap.TotalMisc = idx
	case "totalMaterial":
		colMap.TotalMaterial = idx
	case "totalSubcontract":
		colMap.TotalSubcontract = idx
	case "totalTrucking":
		colMap.TotalTrucking = idx
	case "totalIndCost":
		colMap.TotalIndCost = idx
	case "totalBond":
		colMap.TotalBond = idx
	case "totalOH":
		colMap.TotalOH = idx
	case "totalProfit":
		colMap.TotalProfit = idx
	}
}

// buildBidItems builds the job items with hierarchy using single-pass algorithm.
func buildBidItems(rows [][]string, headerRow int, colMap *bidColumnMap, jobID uuid.UUID) ([]database.InsertBidItemParams, error) {
	var items []database.InsertBidItemParams
	var stack []stackEntry
	tolerance := decimal.NewFromFloat(budgetTolerance)

	itemCounter := 0

	for rowIdx := headerRow + 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]

		// Skip empty rows
		description := getCellValue(row, colMap.Description)
		if description == "" {
			continue
		}

		// Generate UUID for this item
		itemID := uuid.New()
		itemCounter++

		// Parse values
		itemNumber := getCellValue(row, colMap.ItemNumber)
		costMethodRaw := getCellValue(row, colMap.CostMethod)
		productionRateRaw := getCellValue(row, colMap.ProductionRate)
		budgetStr, _ := cleanNumeric(getCellValue(row, colMap.TotalDirectCost))
		budget, _ := decimal.NewFromString(budgetStr)

		// Determine item type and whether it can have children
		var costMethod string
		canHaveChildren := false

		if itemNumber != "" {
			// Pay Item - has item number
			costMethod = "Pay Item"
			canHaveChildren = true
		} else if strings.EqualFold(costMethodRaw, "Detail") {
			// Detail item
			costMethod = "Detail"
			canHaveChildren = true
		} else if strings.EqualFold(costMethodRaw, "Subcontracted") {
			// Subcontracted - no children
			costMethod = "Subcontracted"
			canHaveChildren = false
		} else if costMethodRaw == "" && productionRateRaw != "" {
			// Crew - blank cost method but has production rate
			costMethod = "Crew"
			canHaveChildren = true
		} else {
			// Cost component - use the value directly or default
			if costMethodRaw != "" {
				costMethod = costMethodRaw
			} else {
				costMethod = "Cost"
			}
			canHaveChildren = false
		}

		// STEP 1: Pop any completed parents BEFORE assigning this row
		for len(stack) > 0 {
			top := &stack[len(stack)-1]
			if top.sum.GreaterThanOrEqual(top.target.Sub(tolerance)) {
				stack = stack[:len(stack)-1]
			} else {
				break
			}
		}

		// STEP 2: Assign parent
		var parentID uuid.NullUUID
		if len(stack) > 0 {
			parentID = uuid.NullUUID{UUID: stack[len(stack)-1].item.ID, Valid: true}
			// Add this item's budget to parent's running sum
			stack[len(stack)-1].sum = stack[len(stack)-1].sum.Add(budget)
		}

		// Build the item params
		item := database.InsertBidItemParams{
			ID:              itemID,
			JobID:           jobID,
			ParentID:        parentID,
			SortOrder:       int32(itemCounter),
			ItemNumber:      generateItemNumber(itemNumber, itemCounter),
			Description:     description,
			ScheduledValue:  parseNumericField(row, colMap.TotalBidPrice),
			JobCostID:       toNullString(getCellValue(row, colMap.JobCostID)),
			Budget:          budgetStr,
			Qty:             parseNumericField(row, colMap.Quantity),
			Unit:            toNullString(getCellValue(row, colMap.UM)),
			UnitPrice:       calculateUnitPrice(row, colMap),
			CostMethod:      toNullString(costMethod),
			ProductionRate:  toNullString(getCellValue(row, colMap.ProductionRate)),
			ProductionUnits: toNullString(getCellValue(row, colMap.ProductionMethod)),
			ManHours:        parseNumericField(row, colMap.TotalManHours),
			ProductionHours: parseNumericField(row, colMap.TotalProdHours),
			CrewDays:        parseNumericField(row, colMap.CrewDays),
			Plug:            parseNumericField(row, colMap.TotalPlug),
			Labor:           parseNumericField(row, colMap.TotalLabor),
			Equip:           parseNumericField(row, colMap.TotalEquip),
			Misc:            parseNumericField(row, colMap.TotalMisc),
			Material:        parseNumericField(row, colMap.TotalMaterial),
			Sub:             parseNumericField(row, colMap.TotalSubcontract),
			Trucking:        parseNumericField(row, colMap.TotalTrucking),
			Indirect:        parseNumericField(row, colMap.TotalIndCost),
			Bond:            parseNumericField(row, colMap.TotalBond),
			Overhead:        parseNumericField(row, colMap.TotalOH),
			Profit:          parseNumericField(row, colMap.TotalProfit),
		}

		items = append(items, item)

		// STEP 3: If this item can have children and has budget, push to stack
		if canHaveChildren && budget.GreaterThan(decimal.Zero) {
			stack = append(stack, stackEntry{
				item:   &items[len(items)-1],
				target: budget,
				sum:    decimal.Zero,
			})
		}
	}

	return items, nil
}

// getCellValue safely gets a cell value from a row.
func getCellValue(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// parseNumericField parses a numeric field, returning "0" if empty or invalid.
func parseNumericField(row []string, idx int) string {
	val := getCellValue(row, idx)
	cleaned, err := cleanNumeric(val)
	if err != nil {
		return "0"
	}
	return cleaned
}

// generateItemNumber generates an item number if not provided.
func generateItemNumber(existing string, counter int) string {
	if existing != "" {
		return existing
	}
	return fmt.Sprintf("AUTO-%d", counter)
}

// calculateUnitPrice calculates unit_price from scheduled_value / qty.
func calculateUnitPrice(row []string, colMap *bidColumnMap) string {
	scheduledStr := parseNumericField(row, colMap.TotalBidPrice)
	qtyStr := parseNumericField(row, colMap.Quantity)

	scheduled, err := decimal.NewFromString(scheduledStr)
	if err != nil {
		return "0"
	}

	qty, err := decimal.NewFromString(qtyStr)
	if err != nil || qty.IsZero() {
		return "0"
	}

	return scheduled.Div(qty).StringFixed(4)
}

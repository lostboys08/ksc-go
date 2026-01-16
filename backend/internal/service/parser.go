package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lostboys08/ksc-go/backend/internal/database"
	"github.com/xuri/excelize/v2"
)

// columnMap holds the dynamically detected column indices for static data.
type columnMap struct {
	Item           int // "Item", "Item #"
	Phase          int // "Phase", "Job Cost ID"
	Description    int // "Description", "Description of Work"
	ScheduledValue int // "Scheduled Value"
	Cost           int // "Cost"
	Price          int // "Price"
	Qty            int // "Qty", "Quantity"
	UOM            int // "UOM", "Unit"
	UnitPrice      int // "Unit Price"
	StaticEnd      int // Last static column index
}

// monthColumns represents the column indices for a single month's time series data.
type monthColumns struct {
	Date   time.Time
	Qty    int // QTY column
	Amt    int // AMT column
	Rem    int // REM column
	RemQty int // Rem Qty column
	PctCom int // % Com column
}

// headerPatterns maps field names to possible header variations.
var headerPatterns = map[string][]string{
	"item":           {"item", "item #", "item number"},
	"phase":          {"phase", "job cost id", "job cost", "cost id"},
	"description":    {"description", "desc"},
	"scheduledvalue": {"scheduled value", "scheduled"},
	"cost":           {"cost"},
	"price":          {"price"},
	"qty":            {"qty", "quantity"},
	"uom":            {"uom", "um", "unit"},
	"unitprice":      {"unit price"},
}

// mergedCellMap provides O(1) lookup for merged cell values.
type mergedCellMap struct {
	cellToValue map[string]string
}

// parentItemInfo stores info needed to match parent items between sheets.
type parentItemInfo struct {
	JobItemID      uuid.UUID
	ItemNumber     string
	Description    string
	ScheduledValue string
}

// ParsePayApp parses an Excel file and populates the database.
// It reads the Detail sheet for job items and time series QTY data,
// then reads the SOV sheet to get stored materials for parent items.
func ParsePayApp(ctx context.Context, f *excelize.File, q *database.Queries, jobID uuid.UUID, targetDate time.Time) error {
	targetMonth := time.Date(targetDate.Year(), targetDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Parse Detail sheet - returns parent items for SOV matching
	parentItems, err := parseDetailSheet(ctx, f, q, jobID, targetMonth)
	if err != nil {
		return fmt.Errorf("parsing Detail sheet: %w", err)
	}

	// Parse SOV sheet - updates stored materials for parent items
	err = parseSOVSheet(ctx, f, q, targetMonth, parentItems)
	if err != nil {
		return fmt.Errorf("parsing SOV sheet: %w", err)
	}

	return nil
}

// parseDetailSheet parses the Detail sheet, imports job items and time series data.
// Returns parent item info for matching with SOV sheet.
func parseDetailSheet(ctx context.Context, f *excelize.File, q *database.Queries, jobID uuid.UUID, targetMonth time.Time) ([]parentItemInfo, error) {
	sheetName := findDetailSheet(f)
	if sheetName == "" {
		return nil, fmt.Errorf("could not find Detail sheet")
	}

	mcMap, err := buildMergedCellMap(f, sheetName)
	if err != nil {
		return nil, fmt.Errorf("building merged cell map: %w", err)
	}

	headerRow, colMap, err := findHeadersAndBuildColumnMap(f, sheetName, mcMap)
	if err != nil {
		return nil, fmt.Errorf("finding headers: %w", err)
	}

	monthCols, err := buildMonthColumns(f, sheetName, headerRow, colMap.StaticEnd, mcMap)
	if err != nil {
		return nil, fmt.Errorf("building month columns: %w", err)
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("reading rows: %w", err)
	}

	// Track hierarchy and parent items
	levelToParent := make(map[int]uuid.UUID)
	var parentItems []parentItemInfo

	dataStartRow := headerRow + 2 // Skip both header rows

	for rowIdx := dataStartRow; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		if len(row) == 0 {
			continue
		}

		itemNum := getColValue(row, colMap.Item)
		phase := getColValue(row, colMap.Phase)
		description := getColValue(row, colMap.Description)

		if itemNum == "" && phase == "" && description == "" {
			continue
		}

		outlineLevel, _ := f.GetRowOutlineLevel(sheetName, rowIdx+1)

		var parentID uuid.NullUUID
		if outlineLevel > 0 {
			for level := int(outlineLevel) - 1; level >= 0; level-- {
				if pid, ok := levelToParent[level]; ok {
					parentID = uuid.NullUUID{UUID: pid, Valid: true}
					break
				}
			}
		}

		identifier := itemNum
		if identifier == "" {
			identifier = fmt.Sprintf("row_%d", rowIdx+1)
		}

		scheduledValue, err := cleanNumeric(getColValue(row, colMap.ScheduledValue))
		if err != nil {
			return nil, fmt.Errorf("row %d scheduled value: %w", rowIdx+1, err)
		}

		budget, err := cleanNumeric(getColValue(row, colMap.Cost))
		if err != nil {
			return nil, fmt.Errorf("row %d budget: %w", rowIdx+1, err)
		}

		qty, err := cleanNumeric(getColValue(row, colMap.Qty))
		if err != nil {
			return nil, fmt.Errorf("row %d qty: %w", rowIdx+1, err)
		}

		unitPrice, err := cleanNumeric(getColValue(row, colMap.UnitPrice))
		if err != nil {
			return nil, fmt.Errorf("row %d unit price: %w", rowIdx+1, err)
		}

		params := database.UpsertJobItemParams{
			JobID:          jobID,
			ParentID:       parentID,
			SortOrder:      int32(rowIdx),
			ItemNumber:     identifier,
			Description:    description,
			ScheduledValue: scheduledValue,
			JobCostID:      toNullString(phase),
			Budget:         budget,
			Qty:            qty,
			Unit:           toNullString(getColValue(row, colMap.UOM)),
			UnitPrice:      unitPrice,
		}

		jobItemID, err := q.UpsertJobItem(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("upserting job item at row %d: %w", rowIdx+1, err)
		}

		levelToParent[int(outlineLevel)] = jobItemID

		// Track parent items (level 0 with item number) for SOV matching
		if outlineLevel == 0 && itemNum != "" {
			parentItems = append(parentItems, parentItemInfo{
				JobItemID:      jobItemID,
				ItemNumber:     itemNum,
				Description:    description,
				ScheduledValue: scheduledValue,
			})
		}

		// Import time series data for ALL months
		for _, mc := range monthCols {
			qtyVal := getColValue(row, mc.Qty)
			if qtyVal == "" {
				continue
			}

			cleanedQty, err := cleanNumeric(qtyVal)
			if err != nil {
				return nil, fmt.Errorf("row %d month %s qty: %w", rowIdx+1, mc.Date.Format("Jan-06"), err)
			}

			if mc.Date.Equal(targetMonth) {
				// For target month, use upsert (allows updates)
				err := q.UpsertPayApplication(ctx, database.UpsertPayApplicationParams{
					JobItemID:       jobItemID,
					PayAppMonth:     mc.Date,
					Qty:             cleanedQty,
					StoredMaterials: "0", // Will be updated from SOV sheet
				})
				if err != nil {
					return nil, fmt.Errorf("upserting pay application at row %d: %w", rowIdx+1, err)
				}
			} else {
				// For other months, only insert if not exists (preserve historical data)
				err := q.InsertPayApplicationIfNotExists(ctx, database.InsertPayApplicationIfNotExistsParams{
					JobItemID:       jobItemID,
					PayAppMonth:     mc.Date,
					Qty:             cleanedQty,
					StoredMaterials: "0",
				})
				if err != nil {
					return nil, fmt.Errorf("inserting pay application at row %d: %w", rowIdx+1, err)
				}
			}
		}
	}

	return parentItems, nil
}

// parseSOVSheet parses the SOV sheet to get pay application data for parent items.
// SOV has: ITEM, DESCRIPTION, SCHEDULED VALUE, PREVIOUS, THIS PERIOD, MATERIALS ON SITE, etc.
func parseSOVSheet(ctx context.Context, f *excelize.File, q *database.Queries, targetMonth time.Time, parentItems []parentItemInfo) error {
	sheetName := findSOVSheet(f)
	if sheetName == "" {
		return nil
	}

	mcMap, err := buildMergedCellMap(f, sheetName)
	if err != nil {
		return fmt.Errorf("building merged cell map: %w", err)
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return fmt.Errorf("reading rows: %w", err)
	}

	// Find header row and column positions
	headerRow := -1
	colItem := -1
	colDesc := -1
	colThisPeriod := -1
	colMaterials := -1

	for rowIdx, row := range rows {
		if rowIdx > 10 {
			break
		}

		for colIdx, cell := range row {
			cellLower := strings.ToLower(strings.TrimSpace(cell))
			if cellLower == "" {
				cellLower = strings.ToLower(getCellFlattened(f, sheetName, colIdx+1, rowIdx+1, mcMap))
			}

			if strings.Contains(cellLower, "item") && colItem == -1 {
				colItem = colIdx
				headerRow = rowIdx
			} else if strings.Contains(cellLower, "description") && colDesc == -1 {
				colDesc = colIdx
			} else if strings.Contains(cellLower, "this period") && colThisPeriod == -1 {
				colThisPeriod = colIdx
			} else if strings.Contains(cellLower, "materials") && colMaterials == -1 {
				colMaterials = colIdx
			}
		}

		if colItem >= 0 && colDesc >= 0 && colMaterials >= 0 {
			break
		}
	}

	if headerRow < 0 {
		return nil
	}

	// Also check sub-header row for "THIS PERIOD"
	if colThisPeriod == -1 && headerRow+1 < len(rows) {
		subRow := rows[headerRow+1]
		for colIdx, cell := range subRow {
			cellLower := strings.ToLower(strings.TrimSpace(cell))
			if strings.Contains(cellLower, "this period") {
				colThisPeriod = colIdx
				break
			}
		}
	}

	// Build parent item lookup map
	parentMap := make(map[string]uuid.UUID)
	for _, p := range parentItems {
		key := p.ItemNumber + "|" + strings.ToLower(strings.TrimSpace(p.Description))
		parentMap[key] = p.JobItemID
	}

	// Process SOV data rows
	dataStartRow := headerRow + 2

	for rowIdx := dataStartRow; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		if len(row) == 0 {
			continue
		}

		itemNum := getColValue(row, colItem)
		description := getColValue(row, colDesc)

		if itemNum == "" {
			continue
		}

		// Look up parent item
		key := itemNum + "|" + strings.ToLower(strings.TrimSpace(description))
		jobItemID, found := parentMap[key]
		if !found {
			continue
		}

		// Get values
		thisPeriod := "0"
		if colThisPeriod >= 0 {
			val, err := cleanNumeric(getColValue(row, colThisPeriod))
			if err != nil {
				return fmt.Errorf("SOV row %d this period: %w", rowIdx+1, err)
			}
			thisPeriod = val
		}
		materials := "0"
		if colMaterials >= 0 {
			val, err := cleanNumeric(getColValue(row, colMaterials))
			if err != nil {
				return fmt.Errorf("SOV row %d materials: %w", rowIdx+1, err)
			}
			materials = val
		}

		// Upsert pay application for parent item with SOV data
		err := q.UpsertPayApplication(ctx, database.UpsertPayApplicationParams{
			JobItemID:       jobItemID,
			PayAppMonth:     targetMonth,
			Qty:             thisPeriod,
			StoredMaterials: materials,
		})
		if err != nil {
			return fmt.Errorf("upserting pay application for item %s: %w", itemNum, err)
		}
	}

	return nil
}

// findDetailSheet finds the Detail sheet by name.
func findDetailSheet(f *excelize.File) string {
	for _, name := range f.GetSheetList() {
		if strings.EqualFold(name, "detail") {
			return name
		}
	}
	sheets := f.GetSheetList()
	if len(sheets) >= 2 {
		return sheets[1]
	}
	return ""
}

// findSOVSheet finds the SOV sheet by name.
func findSOVSheet(f *excelize.File) string {
	for _, name := range f.GetSheetList() {
		if strings.EqualFold(name, "sov") {
			return name
		}
	}
	sheets := f.GetSheetList()
	if len(sheets) >= 1 {
		return sheets[0]
	}
	return ""
}

// buildMergedCellMap creates a lookup map that flattens merged cells.
func buildMergedCellMap(f *excelize.File, sheetName string) (*mergedCellMap, error) {
	mcMap := &mergedCellMap{
		cellToValue: make(map[string]string),
	}

	mergedCells, err := f.GetMergeCells(sheetName)
	if err != nil {
		return nil, err
	}

	for _, mc := range mergedCells {
		startCol, startRow, err := excelize.CellNameToCoordinates(mc.GetStartAxis())
		if err != nil {
			continue
		}
		endCol, endRow, err := excelize.CellNameToCoordinates(mc.GetEndAxis())
		if err != nil {
			continue
		}

		value := mc.GetCellValue()
		for row := startRow; row <= endRow; row++ {
			for col := startCol; col <= endCol; col++ {
				cellRef, _ := excelize.CoordinatesToCellName(col, row)
				mcMap.cellToValue[cellRef] = value
			}
		}
	}

	return mcMap, nil
}

// getCellFlattened gets a cell value, checking merged cells first.
func getCellFlattened(f *excelize.File, sheetName string, col, row int, mcMap *mergedCellMap) string {
	cellRef, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return ""
	}

	if val, ok := mcMap.cellToValue[cellRef]; ok {
		return strings.TrimSpace(val)
	}

	val, _ := f.GetCellValue(sheetName, cellRef)
	return strings.TrimSpace(val)
}

// findHeadersAndBuildColumnMap locates the header row and maps static column positions.
func findHeadersAndBuildColumnMap(f *excelize.File, sheetName string, mcMap *mergedCellMap) (int, *columnMap, error) {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return 0, nil, err
	}

	for rowIdx, row := range rows {
		if rowIdx > 10 {
			break
		}

		score := 0
		for colIdx := 0; colIdx < len(row) && colIdx < 15; colIdx++ {
			cellVal := strings.ToLower(strings.TrimSpace(row[colIdx]))
			if cellVal == "" {
				cellVal = strings.ToLower(getCellFlattened(f, sheetName, colIdx+1, rowIdx+1, mcMap))
			}
			if matchesAnyPattern(cellVal) {
				score++
			}
		}

		if score >= 3 {
			colMap := buildColumnMap(f, sheetName, rowIdx, row, mcMap)
			return rowIdx, colMap, nil
		}
	}

	return 0, nil, fmt.Errorf("header row not found")
}

// matchesAnyPattern checks if a value matches any header pattern.
func matchesAnyPattern(value string) bool {
	for _, variations := range headerPatterns {
		for _, pattern := range variations {
			if strings.Contains(value, pattern) {
				return true
			}
		}
	}
	return false
}

// buildColumnMap builds the static column map from the header row.
func buildColumnMap(f *excelize.File, sheetName string, rowIdx int, row []string, mcMap *mergedCellMap) *columnMap {
	colMap := &columnMap{
		Item:           -1,
		Phase:          -1,
		Description:    -1,
		ScheduledValue: -1,
		Cost:           -1,
		Price:          -1,
		Qty:            -1,
		UOM:            -1,
		UnitPrice:      -1,
		StaticEnd:      -1,
	}

	for colIdx := 0; colIdx < len(row); colIdx++ {
		cellVal := strings.ToLower(strings.TrimSpace(row[colIdx]))
		if cellVal == "" {
			cellVal = strings.ToLower(getCellFlattened(f, sheetName, colIdx+1, rowIdx+1, mcMap))
		}

		matched := false
		if matchesPatterns(cellVal, headerPatterns["item"]) && colMap.Item == -1 {
			colMap.Item = colIdx
			matched = true
		} else if matchesPatterns(cellVal, headerPatterns["phase"]) && colMap.Phase == -1 {
			colMap.Phase = colIdx
			matched = true
		} else if matchesPatterns(cellVal, headerPatterns["description"]) && colMap.Description == -1 {
			colMap.Description = colIdx
			matched = true
		} else if matchesPatterns(cellVal, headerPatterns["scheduledvalue"]) && colMap.ScheduledValue == -1 {
			colMap.ScheduledValue = colIdx
			matched = true
		} else if matchesPatterns(cellVal, headerPatterns["cost"]) && colMap.Cost == -1 {
			colMap.Cost = colIdx
			matched = true
		} else if matchesPatterns(cellVal, headerPatterns["price"]) && colMap.Price == -1 {
			colMap.Price = colIdx
			matched = true
		} else if matchesPatterns(cellVal, headerPatterns["qty"]) && colMap.Qty == -1 {
			colMap.Qty = colIdx
			matched = true
		} else if matchesPatterns(cellVal, headerPatterns["uom"]) && colMap.UOM == -1 {
			colMap.UOM = colIdx
			matched = true
		} else if matchesPatterns(cellVal, headerPatterns["unitprice"]) && colMap.UnitPrice == -1 {
			colMap.UnitPrice = colIdx
			matched = true
		}

		if matched {
			colMap.StaticEnd = colIdx
		}
	}

	return colMap
}

// matchesPatterns checks if a value matches any of the given patterns.
func matchesPatterns(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(value, pattern) {
			return true
		}
	}
	return false
}

// buildMonthColumns identifies the time series columns for each month.
func buildMonthColumns(f *excelize.File, sheetName string, headerRow, staticEnd int, mcMap *mergedCellMap) ([]monthColumns, error) {
	var months []monthColumns

	cols, err := f.GetCols(sheetName)
	if err != nil {
		return nil, err
	}
	totalCols := len(cols)

	startCol := staticEnd + 1
	if startCol < 0 {
		startCol = 9
	}

	for colIdx := startCol; colIdx < totalCols; {
		monthStr := getCellFlattened(f, sheetName, colIdx+1, headerRow+1, mcMap)

		monthDate, err := parseMonthHeader(monthStr)
		if err != nil {
			colIdx++
			continue
		}

		mc := monthColumns{
			Date:   monthDate,
			Qty:    colIdx,
			Amt:    colIdx + 1,
			Rem:    colIdx + 2,
			RemQty: colIdx + 3,
			PctCom: colIdx + 4,
		}
		months = append(months, mc)

		colIdx += 5
	}

	return months, nil
}

// parseMonthHeader parses month header strings like "Dec-25", "December 2025", etc.
func parseMonthHeader(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	formats := []string{
		"Jan-06",
		"January-06",
		"Jan 06",
		"January 06",
		"Jan-2006",
		"January-2006",
		"Jan 2006",
		"January 2006",
		"01/2006",
		"1/2006",
		"2006-01",
		"2006-1",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}

// getColValue safely retrieves a value from the row.
func getColValue(row []string, colIdx int) string {
	if colIdx >= 0 && colIdx < len(row) {
		return strings.TrimSpace(row[colIdx])
	}
	return ""
}

// toNullString converts a string to sql.NullString.
func toNullString(s string) sql.NullString {
	s = strings.TrimSpace(s)
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}

// cleanNumeric strips currency symbols, commas, and whitespace from numeric strings.
// Returns an error for Excel errors like #REF!, #DIV/0!, #N/A, #VALUE!
func cleanNumeric(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "0", nil
	}
	if strings.HasPrefix(s, "#") {
		return "", fmt.Errorf("Excel error value found: %s", s)
	}
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "%", "")
	if s == "" {
		return "0", nil
	}
	return s, nil
}

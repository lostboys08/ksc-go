package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lostboys08/ksc-go/backend/internal/database"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

// SheetResult contains the result of processing a single sheet.
type SheetResult struct {
	SheetName     string `json:"sheetName"`
	RowsProcessed int    `json:"rowsProcessed"`
	RowsInserted  int    `json:"rowsInserted"`
	RowsSkipped   int    `json:"rowsSkipped"`
	Error         string `json:"error,omitempty"`
}

// UploadResult contains the result of a file upload operation.
type UploadResult struct {
	Success       bool          `json:"success"`
	Message       string        `json:"message"`
	Filename      string        `json:"filename,omitempty"`
	RowsProcessed int           `json:"rowsProcessed,omitempty"`
	SheetResults  []SheetResult `json:"sheetResults,omitempty"`
}

// ImportPayApplication imports a pay application Excel file for a specific job and month.
func ImportPayApplication(ctx context.Context, f *excelize.File, q *database.Queries, jobNumber, jobName string, targetDate time.Time) (*UploadResult, error) {
	// Get or create job
	jobID, err := getOrCreateJob(ctx, q, jobNumber, jobName)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create job: %w", err)
	}

	// Parse and import
	err = ParsePayApp(ctx, f, q, jobID, targetDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pay application: %w", err)
	}

	return &UploadResult{
		Success: true,
		Message: fmt.Sprintf("Successfully imported pay application for job %s, %s", jobNumber, targetDate.Format("January 2006")),
	}, nil
}

// ImportCostLedger imports a cost ledger Excel file, processing all sheets.
func ImportCostLedger(ctx context.Context, f *excelize.File, q *database.Queries) (*UploadResult, error) {
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("Excel file has no sheets")
	}

	var sheetResults []SheetResult
	totalInserted := 0
	totalSkipped := 0

	for _, sheetName := range sheets {
		result := processSheet(ctx, f, q, sheetName)
		sheetResults = append(sheetResults, result)
		totalInserted += result.RowsInserted
		totalSkipped += result.RowsSkipped
	}

	return &UploadResult{
		Success:       true,
		Message:       fmt.Sprintf("Processed %d sheets: %d inserted, %d skipped", len(sheets), totalInserted, totalSkipped),
		RowsProcessed: totalInserted,
		SheetResults:  sheetResults,
	}, nil
}

// processSheet processes a single sheet from the cost ledger workbook.
func processSheet(ctx context.Context, f *excelize.File, q *database.Queries, sheetName string) SheetResult {
	result := SheetResult{SheetName: sheetName}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get rows: %v", err)
		return result
	}

	if len(rows) < 2 {
		result.Error = "sheet has no data rows"
		return result
	}

	result.RowsProcessed = len(rows) - 1 // exclude header

	for i, row := range rows {
		if i == 0 {
			continue
		}

		if len(row) < 7 {
			result.RowsSkipped++
			continue
		}

		job := strings.TrimSpace(row[0])
		phase := strings.TrimSpace(row[1])
		cat := strings.TrimSpace(row[2])
		transactionType := strings.TrimSpace(row[3])
		transactionDateStr := strings.TrimSpace(row[4])
		amountStr := strings.TrimSpace(row[6])

		var transactionDate sql.NullTime
		if transactionDateStr != "" {
			t, err := parseLedgerDate(transactionDateStr)
			if err == nil {
				transactionDate = sql.NullTime{Time: t, Valid: true}
			}
		}

		amount := decimal.Zero
		amountClean := strings.ReplaceAll(amountStr, ",", "")
		if amountClean != "" {
			d, err := decimal.NewFromString(amountClean)
			if err == nil {
				amount = d
			}
		}

		if amount.IsZero() {
			result.RowsSkipped++
			continue
		}

		hashInput := fmt.Sprintf("%s|%s|%s|%s|%s|%s", job, phase, cat, transactionType, transactionDateStr, amountStr)
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(hashInput)))

		params := database.InsertJobCostLedgerParams{
			ID:              hash,
			Job:             job,
			Phase:           toNullString(phase),
			Cat:             toNullString(cat),
			TransactionType: toNullString(transactionType),
			TransactionDate: transactionDate,
			Amount:          amount.String(),
		}

		err := q.InsertJobCostLedger(ctx, params)
		if err != nil {
			result.RowsSkipped++
			continue
		}
		result.RowsInserted++
	}

	return result
}

func getOrCreateJob(ctx context.Context, q *database.Queries, jobNumber, jobName string) (uuid.UUID, error) {
	job, err := q.GetJobByNumber(ctx, jobNumber)
	if err == nil {
		return job.ID, nil
	}

	if jobName == "" {
		jobName = jobNumber
	}

	return q.UpsertJob(ctx, database.UpsertJobParams{
		JobNumber: jobNumber,
		JobName:   jobName,
	})
}

func parseLedgerDate(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"01/02/2006",
		"1/2/2006",
		"01-02-2006",
		"2006/01/02",
		"Jan 2, 2006",
		"January 2, 2006",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date '%s'", s)
}

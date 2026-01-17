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

// UploadResult contains the result of a file upload operation.
type UploadResult struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	Filename     string `json:"filename,omitempty"`
	RowsProcessed int   `json:"rowsProcessed,omitempty"`
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

// ImportCostLedger imports a cost ledger Excel file.
func ImportCostLedger(ctx context.Context, f *excelize.File, q *database.Queries) (*UploadResult, error) {
	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("Excel file has no data rows")
	}

	inserted := 0
	skipped := 0

	for i, row := range rows {
		if i == 0 {
			continue
		}

		if len(row) < 7 {
			skipped++
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
			skipped++
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
			skipped++
			continue
		}
		inserted++
	}

	return &UploadResult{
		Success:       true,
		Message:       fmt.Sprintf("Imported %d records, skipped %d", inserted, skipped),
		RowsProcessed: inserted,
	}, nil
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

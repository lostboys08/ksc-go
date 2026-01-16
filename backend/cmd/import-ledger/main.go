package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"

	"github.com/lostboys08/ksc-go/backend/internal/database"
)

func main() {
	filePath := flag.String("file", "", "Path to the Excel file to import")
	dbURL := flag.String("db", "postgres://ksc:password@localhost:5432/ksc_data?sslmode=disable", "Database connection URL")

	flag.Parse()

	if *filePath == "" {
		fmt.Println("Error: -file is required")
		flag.Usage()
		os.Exit(1)
	}

	// Connect to database
	db, err := sql.Open("postgres", *dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	queries := database.New(db)
	ctx := context.Background()

	// Open Excel file
	f, err := excelize.OpenFile(*filePath)
	if err != nil {
		log.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Get the first sheet
	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Fatalf("Failed to get rows: %v", err)
	}

	if len(rows) < 2 {
		log.Fatal("Excel file has no data rows")
	}

	// Expected columns: Job, Phase, Cat, Transaction Type, Transaction Date, Accounting Date, Amount, Description
	// We keep: Job (0), Phase (1), Cat (2), Transaction Type (3), Transaction Date (4), Amount (6)
	// We drop: Accounting Date (5), Description (7)

	log.Printf("Found %d rows (including header)", len(rows))

	inserted := 0
	skipped := 0

	for i, row := range rows {
		if i == 0 {
			// Skip header row
			continue
		}

		if len(row) < 7 {
			log.Printf("Row %d: skipping, not enough columns (%d)", i+1, len(row))
			skipped++
			continue
		}

		job := strings.TrimSpace(row[0])
		phase := strings.TrimSpace(row[1])
		cat := strings.TrimSpace(row[2])
		transactionType := strings.TrimSpace(row[3])
		transactionDateStr := strings.TrimSpace(row[4])
		amountStr := strings.TrimSpace(row[6])

		// Parse transaction date
		var transactionDate sql.NullTime
		if transactionDateStr != "" {
			t, err := parseDate(transactionDateStr)
			if err != nil {
				log.Printf("Row %d: failed to parse date '%s': %v", i+1, transactionDateStr, err)
			} else {
				transactionDate = sql.NullTime{Time: t, Valid: true}
			}
		}

		// Parse amount (remove commas first)
		amount := decimal.Zero
		amountClean := strings.ReplaceAll(amountStr, ",", "")
		if amountClean != "" {
			d, err := decimal.NewFromString(amountClean)
			if err != nil {
				log.Printf("Row %d: failed to parse amount '%s': %v", i+1, amountStr, err)
			} else {
				amount = d
			}
		}

		// Skip records with zero amount
		if amount.IsZero() {
			skipped++
			continue
		}

		// Create hash from row content
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

		err := queries.InsertJobCostLedger(ctx, params)
		if err != nil {
			log.Printf("Row %d: failed to insert: %v", i+1, err)
			skipped++
			continue
		}
		inserted++
	}

	log.Printf("Import completed: %d inserted, %d skipped", inserted, skipped)
}

func parseDate(s string) (time.Time, error) {
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

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/xuri/excelize/v2"

	"github.com/lostboys08/ksc-go/backend/internal/database"
	"github.com/lostboys08/ksc-go/backend/internal/service"
)

func main() {
	// Command line flags
	filePath := flag.String("file", "", "Path to the Excel file to import")
	jobNumber := flag.String("job", "", "Job number to import data for (will create if not exists)")
	jobName := flag.String("name", "", "Job name (optional, used when creating new job)")
	targetDateStr := flag.String("date", "", "Target month to import (format: 2006-01 or January 2006)")
	dbURL := flag.String("db", "postgres://ksc:password@localhost:5432/ksc_data?sslmode=disable", "Database connection URL")

	flag.Parse()

	// Validate required flags
	if *filePath == "" {
		fmt.Println("Error: -file is required")
		flag.Usage()
		os.Exit(1)
	}
	if *jobNumber == "" {
		fmt.Println("Error: -job is required")
		flag.Usage()
		os.Exit(1)
	}
	if *targetDateStr == "" {
		fmt.Println("Error: -date is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse target date
	targetDate, err := parseDate(*targetDateStr)
	if err != nil {
		log.Fatalf("Invalid date format: %v", err)
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

	// Get or create job
	jobID, err := getOrCreateJob(ctx, queries, *jobNumber, *jobName)
	if err != nil {
		log.Fatalf("Failed to get/create job: %v", err)
	}
	log.Printf("Using job ID: %s", jobID)

	// Open Excel file
	f, err := excelize.OpenFile(*filePath)
	if err != nil {
		log.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Parse and import
	log.Printf("Importing data for %s...", targetDate.Format("January 2006"))
	err = service.ParsePayApp(ctx, f, queries, jobID, targetDate)
	if err != nil {
		log.Fatalf("Failed to parse pay application: %v", err)
	}

	log.Println("Import completed successfully!")
}

func parseDate(s string) (time.Time, error) {
	formats := []string{
		"2006-01",
		"January 2006",
		"Jan 2006",
		"01/2006",
		"1/2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date '%s' - use format like '2006-01' or 'January 2006'", s)
}

func getOrCreateJob(ctx context.Context, q *database.Queries, jobNumber, jobName string) (uuid.UUID, error) {
	// Try to find existing job
	job, err := q.GetJobByNumber(ctx, jobNumber)
	if err == nil {
		return job.ID, nil
	}

	// Create new job if not found
	if jobName == "" {
		jobName = jobNumber // Use job number as name if not provided
	}

	log.Printf("Creating new job: %s (%s)", jobNumber, jobName)
	return q.UpsertJob(ctx, database.UpsertJobParams{
		JobNumber: jobNumber,
		JobName:   jobName,
	})
}

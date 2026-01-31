package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/xuri/excelize/v2"

	"github.com/lostboys08/ksc-go/backend/internal/database"
	"github.com/lostboys08/ksc-go/backend/internal/service"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func runMigrations(db *sql.DB) error {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	// Sort files by name to ensure order
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	for _, file := range files {
		content, err := migrationsFS.ReadFile("migrations/" + file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file, err)
		}

		log.Printf("Running migration: %s", file)
		_, err = db.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed to run migration %s: %w", file, err)
		}
	}

	return nil
}

func main() {
	log.Println("Backend starting...")

	dbURL := os.Getenv("DATABASE_URL")
	log.Println("DATABASE_URL set:", dbURL != "")
	if dbURL == "" {
		dbURL = "postgres://ksc:password@localhost:5432/ksc_data?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Cannot connect to database:", err)
	}
	log.Println("Connected to database")

	// Run migrations
	if err := runMigrations(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}
	log.Println("Migrations completed")

	queries := database.New(db)

	// Example: list all jobs
	jobs, err := queries.GetAllJobs(context.Background())
	if err != nil {
		log.Println("Error fetching jobs:", err)
	} else {
		log.Printf("Found %d jobs in database", len(jobs))
	}

	log.Println("Server starting on :8080")

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	http.HandleFunc("/api/upload", handleUpload(queries))
	http.HandleFunc("/api/jobs", handleGetJobs(queries))
	http.HandleFunc("/api/jobs/cost-over-time", handleGetCostOverTime(queries))
	http.HandleFunc("/api/jobs/cost-performance-index", handleGetCostPerformanceIndex(queries))
	http.HandleFunc("/api/jobs/over-budget-phases", handleGetOverBudgetPhases(queries))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleUpload(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse multipart form (max 32MB)
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to get file: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		uploadType := r.FormValue("type")
		if uploadType == "" {
			http.Error(w, "Upload type is required", http.StatusBadRequest)
			return
		}

		// Open Excel file from uploaded content
		f, err := excelize.OpenReader(file)
		if err != nil {
			http.Error(w, "Failed to parse Excel file: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer f.Close()

		ctx := context.Background()
		var result *service.UploadResult

		switch uploadType {
		case "pay-application":
			jobNumber := r.FormValue("jobNumber")
			dateStr := r.FormValue("date")

			if jobNumber == "" {
				http.Error(w, "Job number is required for pay application import", http.StatusBadRequest)
				return
			}
			if dateStr == "" {
				http.Error(w, "Date is required for pay application import", http.StatusBadRequest)
				return
			}

			targetDate, err := parseTargetDate(dateStr)
			if err != nil {
				http.Error(w, "Invalid date format: "+err.Error(), http.StatusBadRequest)
				return
			}

			result, err = service.ImportPayApplication(ctx, f, queries, jobNumber, "", targetDate)
			if err != nil {
				http.Error(w, "Import failed: "+err.Error(), http.StatusInternalServerError)
				return
			}

		case "cost-ledger":
			result, err = service.ImportCostLedger(ctx, f, queries)
			if err != nil {
				http.Error(w, "Import failed: "+err.Error(), http.StatusInternalServerError)
				return
			}

		case "bid":
			jobNumber := r.FormValue("jobNumber")
			jobName := r.FormValue("jobName")

			if jobNumber == "" {
				http.Error(w, "Job number is required for bid import", http.StatusBadRequest)
				return
			}

			result, err = service.ImportBid(ctx, f, queries, jobNumber, jobName)
			if err != nil {
				http.Error(w, "Import failed: "+err.Error(), http.StatusInternalServerError)
				return
			}

		default:
			http.Error(w, "Unknown upload type: "+uploadType, http.StatusBadRequest)
			return
		}

		result.Filename = header.Filename

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func parseTargetDate(s string) (time.Time, error) {
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

	return time.Time{}, fmt.Errorf("unable to parse date '%s' - use format like '2006-01'", s)
}

func handleGetJobs(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		jobs, err := queries.GetAllJobs(context.Background())
		if err != nil {
			http.Error(w, "Failed to fetch jobs: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobs)
	}
}

type MonthlyPerformanceResponse struct {
	Month            string `json:"month"`
	CostTotal        int64  `json:"cost_total"`
	PayAppTotal      int64  `json:"pay_app_total"`
	CumulativeCost   int64  `json:"cumulative_cost"`
	CumulativePayApp int64  `json:"cumulative_pay_app"`
}

func handleGetCostOverTime(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		jobNumber := r.URL.Query().Get("job")
		if jobNumber == "" {
			http.Error(w, "job query parameter is required", http.StatusBadRequest)
			return
		}

		rows, err := queries.GetMonthlyPerformance(context.Background(), jobNumber)
		if err != nil {
			http.Error(w, "Failed to fetch performance data: "+err.Error(), http.StatusInternalServerError)
			return
		}

		response := make([]MonthlyPerformanceResponse, 0, len(rows))
		for _, row := range rows {
			response = append(response, MonthlyPerformanceResponse{
				Month:            row.Month.Format("2006-01"),
				CostTotal:        row.CostTotal,
				PayAppTotal:      row.PayAppTotal,
				CumulativeCost:   row.CumulativeCost,
				CumulativePayApp: row.CumulativePayApp,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

type CPIResponse struct {
	Month             string `json:"month"`
	Budget            int64  `json:"budget"`
	TotalScheduledQty string `json:"total_scheduled_qty"`
	CumulativeQty     string `json:"cumulative_qty"`
	PercentComplete   string `json:"percent_complete"`
	EarnedValue       int64  `json:"earned_value"`
	ActualCost        int64  `json:"actual_cost"`
	CPI               string `json:"cpi"`
}

func handleGetCostPerformanceIndex(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		jobNumber := r.URL.Query().Get("job")
		if jobNumber == "" {
			http.Error(w, "job query parameter is required", http.StatusBadRequest)
			return
		}

		rows, err := queries.GetCostPerformanceIndex(context.Background(), jobNumber)
		if err != nil {
			http.Error(w, "Failed to fetch CPI data: "+err.Error(), http.StatusInternalServerError)
			return
		}

		response := make([]CPIResponse, 0, len(rows))
		for _, row := range rows {
			response = append(response, CPIResponse{
				Month:             row.Month.Format("2006-01"),
				Budget:            row.Budget,
				TotalScheduledQty: row.TotalScheduledQty,
				CumulativeQty:     row.CumulativeQty,
				PercentComplete:   row.PercentComplete,
				EarnedValue:       row.EarnedValue,
				ActualCost:        row.ActualCost,
				CPI:               row.Cpi,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

type OverBudgetPhaseResponse struct {
	Phase       string `json:"phase"`
	Description string `json:"description"`
	Budget      int64  `json:"budget"`
	ActualCost  int64  `json:"actual_cost"`
	Variance    int64  `json:"variance"`
}

func handleGetOverBudgetPhases(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		jobNumber := r.URL.Query().Get("job")
		if jobNumber == "" {
			http.Error(w, "job query parameter is required", http.StatusBadRequest)
			return
		}

		rows, err := queries.GetOverBudgetPhases(context.Background(), jobNumber)
		if err != nil {
			http.Error(w, "Failed to fetch over-budget phases: "+err.Error(), http.StatusInternalServerError)
			return
		}

		response := make([]OverBudgetPhaseResponse, 0, len(rows))
		for _, row := range rows {
			phase := ""
			if row.Phase.Valid {
				phase = row.Phase.String
			}
			response = append(response, OverBudgetPhaseResponse{
				Phase:       phase,
				Description: string(row.Description),
				Budget:      row.Budget,
				ActualCost:  row.ActualCost,
				Variance:    row.Variance,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

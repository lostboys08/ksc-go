package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	_ "github.com/lib/pq"

	"github.com/lostboys08/ksc-go/backend/internal/database"
)

func main() {
	db, err := sql.Open("postgres", "postgres://ksc:password@localhost:5432/ksc_data?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Cannot connect to database:", err)
	}
	log.Println("Connected to database")

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
	log.Fatal(http.ListenAndServe(":8080", nil))
}

package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/poll/api/internal/adapters/repository/postgres"
	"github.com/poll/api/internal/core/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	var dbHost, dbPort, dbUser, dbPass, dbName string

	flag.StringVar(&dbHost, "db-host", os.Getenv("POSTGRES_HOST"), "Database host")
	flag.StringVar(&dbPort, "db-port", os.Getenv("POSTGRES_PORT"), "Database port")
	flag.StringVar(&dbUser, "db-user", os.Getenv("POSTGRES_USER"), "Database user")
	flag.StringVar(&dbPass, "db-pass", os.Getenv("POSTGRES_PASSWORD"), "Database password")
	flag.StringVar(&dbName, "db-name", os.Getenv("POSTGRES_DB"), "Database name")
	flag.Parse()

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	// Initialize Repositories
	pollRepo := postgres.NewPollRepository(db)
	resultRepo := postgres.NewPollResultRepository(db)

	// Initialize Service
	summaryService := services.NewSummaryService(pollRepo, resultRepo)

	// Use a timeout for the job execution to prevent it from hanging indefinitely
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Println("Starting vote summarization job...")

	if err := summaryService.SummarizeAllVotes(ctx); err != nil {
		log.Fatalf("Error summarizing votes: %v", err)
	}

	log.Println("Vote summarization completed successfully.")
}

package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/poll/api/internal/adapters/handler/http"
	"github.com/poll/api/internal/adapters/repository/postgres"
	"github.com/poll/api/internal/core/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPass := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	pollRepo := postgres.NewPollRepository(db)
	voteRepo := postgres.NewVoteRepository(db)
	resultRepo := postgres.NewPollResultRepository(db)
	pollService := services.NewPollService(pollRepo, resultRepo)
	voteService := services.NewVoteService(pollRepo, voteRepo)

	pollHandler := http.NewPollHandler(pollService)
	voteHandler := http.NewVoteHandler(voteService)
	handler := http.NewHandler(pollHandler, voteHandler)

	server := &stdhttp.Server{Addr: "0.0.0.0:8080", Handler: handler}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		fmt.Println("Server is starting on :8080")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	fmt.Println("Gracefully shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}
}

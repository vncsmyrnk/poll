package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	pollHttp "github.com/poll/api/internal/adapters/handler/http"
	pollRepo "github.com/poll/api/internal/adapters/repository/postgres"
	"github.com/poll/api/internal/core/domain"
	"github.com/poll/api/internal/core/ports"
	"github.com/poll/api/internal/core/services"
)

type TestApp struct {
	DB          *sql.DB
	Server      *httptest.Server
	Client      *http.Client
	SummarySvc  ports.SummaryService
	DBContainer testcontainers.Container
}

func setupTestApp(t *testing.T) *TestApp {
	ctx := context.Background()
	dbContainer, dbURL, err := setupPostgresContainer(ctx)
	require.NoError(t, err)

	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err)

	err = applyMigrations(db)
	require.NoError(t, err)

	repo := pollRepo.NewPollRepository(db)
	voteRepo := pollRepo.NewVoteRepository(db)
	resultRepo := pollRepo.NewPollResultRepository(db)

	svc := services.NewPollService(repo, resultRepo)
	voteSvc := services.NewVoteService(repo, voteRepo)
	summarySvc := services.NewSummaryService(repo, resultRepo)

	handler := pollHttp.NewPollHandler(svc)
	voteHandler := pollHttp.NewVoteHandler(voteSvc)
	router := pollHttp.NewHandler(handler, voteHandler)

	server := httptest.NewServer(router)

	return &TestApp{
		DB:          db,
		Server:      server,
		Client:      server.Client(),
		SummarySvc:  summarySvc,
		DBContainer: dbContainer,
	}
}

func (app *TestApp) Teardown(t *testing.T) {
	app.Server.Close()
	app.DB.Close()
	if err := app.DBContainer.Terminate(context.Background()); err != nil {
		t.Logf("failed to terminate container: %v", err)
	}
}

// TestPollFlow tests the basic lifecycle: Create Poll -> Get Poll -> Vote -> Prevent Duplicate Vote
func TestPollFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	app := setupTestApp(t)
	defer app.Teardown(t)

	// Step 1: Create a Poll
	createPayload := map[string]interface{}{
		"title":       "Flow Test Poll",
		"description": "Testing the basic flow",
		"options":     []string{"Option A", "Option B"},
	}
	body, _ := json.Marshal(createPayload)

	resp, err := app.Client.Post(app.Server.URL+"/api/polls", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdPoll domain.Poll
	err = json.NewDecoder(resp.Body).Decode(&createdPoll)
	require.NoError(t, err)
	resp.Body.Close()

	assert.NotEqual(t, uuid.Nil, createdPoll.ID)
	assert.Equal(t, createPayload["title"], createdPoll.Title)
	assert.Len(t, createdPoll.Options, 2)

	// Step 2: Get the Poll
	resp, err = app.Client.Get(fmt.Sprintf("%s/api/polls/%s", app.Server.URL, createdPoll.ID))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var fetchedPoll domain.Poll
	err = json.NewDecoder(resp.Body).Decode(&fetchedPoll)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, createdPoll.ID, fetchedPoll.ID)

	// Step 3: Cast a Vote
	votePayload := map[string]interface{}{
		"option_id": fetchedPoll.Options[0].ID,
	}
	voteBody, _ := json.Marshal(votePayload)

	resp, err = app.Client.Post(fmt.Sprintf("%s/api/polls/%s/votes", app.Server.URL, createdPoll.ID), "application/json", bytes.NewReader(voteBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Step 4: Duplicate Vote
	resp, err = app.Client.Post(fmt.Sprintf("%s/api/polls/%s/votes", app.Server.URL, createdPoll.ID), "application/json", bytes.NewReader(voteBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, resp.StatusCode)
}

// TestVoteSummarization tests that the worker logic correctly aggregates votes and the API returns stats.
func TestVoteSummarization(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	app := setupTestApp(t)
	defer app.Teardown(t)

	// 1. Create Poll
	createPayload := map[string]interface{}{
		"title":       "Stats Test Poll",
		"description": "Testing aggregation",
		"options":     []string{"Opt1", "Opt2", "Opt3"},
	}
	body, _ := json.Marshal(createPayload)
	resp, err := app.Client.Post(app.Server.URL+"/api/polls", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	var poll domain.Poll
	json.NewDecoder(resp.Body).Decode(&poll)
	resp.Body.Close()

	opt1 := poll.Options[0].ID
	opt2 := poll.Options[1].ID

	// 2. Insert Votes Directly into DB (to simulate multiple users)
	// Opt1: 2 votes
	// Opt2: 1 vote
	// Opt3: 0 votes
	_, err = app.DB.Exec(`INSERT INTO votes (poll_id, option_id, voter_ip) VALUES ($1, $2, '1.1.1.1')`, poll.ID, opt1)
	require.NoError(t, err)
	_, err = app.DB.Exec(`INSERT INTO votes (poll_id, option_id, voter_ip) VALUES ($1, $2, '1.1.1.2')`, poll.ID, opt1)
	require.NoError(t, err)
	_, err = app.DB.Exec(`INSERT INTO votes (poll_id, option_id, voter_ip) VALUES ($1, $2, '1.1.1.3')`, poll.ID, opt2)
	require.NoError(t, err)

	// 3. Run Summarization
	err = app.SummarySvc.SummarizeAllVotes(context.Background())
	require.NoError(t, err)

	// 4. Check API Results
	resp, err = app.Client.Get(fmt.Sprintf("%s/api/polls/%s", app.Server.URL, poll.ID))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var statsPoll domain.Poll
	json.NewDecoder(resp.Body).Decode(&statsPoll)
	resp.Body.Close()

	for _, o := range statsPoll.Options {
		if o.ID == opt1 {
			assert.Equal(t, int64(2), o.VoteCount)
			assert.InDelta(t, 66.66, o.Percentage, 1.0)
		} else if o.ID == opt2 {
			assert.Equal(t, int64(1), o.VoteCount)
			assert.InDelta(t, 33.33, o.Percentage, 1.0)
		} else {
			assert.Equal(t, int64(0), o.VoteCount)
			assert.Equal(t, 0.0, o.Percentage)
		}
	}
}

func setupPostgresContainer(ctx context.Context) (testcontainers.Container, string, error) {
	dbName := "testdb"
	user := "user"
	password := "password"

	pgContainer, err := postgres.Run(ctx, "postgres:15-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(user),
		postgres.WithPassword(password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to start postgres container: %w", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, "", err
	}

	return pgContainer, connStr, nil
}

func applyMigrations(db *sql.DB) error {
	dirPath := "../../internal/adapters/repository/postgres/migrations"

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), "up.sql") {
			continue
		}

		fullPath := filepath.Join(dirPath, entry.Name())
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", entry.Name(), err)
		}

		_, err = db.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}

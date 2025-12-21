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

// TestListPolls tests pagination and fuzzy search
func TestListPolls(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	app := setupTestApp(t)
	defer app.Teardown(t)

	// Create 15 polls
	// 5 "Alpha", 5 "Beta", 5 "Gamma"
	prefixes := []string{"Alpha", "Beta", "Gamma"}
	for _, prefix := range prefixes {
		for i := 1; i <= 5; i++ {
			payload := map[string]interface{}{
				"title":       fmt.Sprintf("%s Poll %d", prefix, i),
				"description": "Desc",
				"options":     []string{"A", "B"},
			}
			body, _ := json.Marshal(payload)
			// Introduce slight delay to ensure deterministic ordering by CreatedAt
			time.Sleep(10 * time.Millisecond)
			resp, err := app.Client.Post(app.Server.URL+"/api/polls", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			resp.Body.Close()
		}
	}

	// 1. Test Pagination (Page 1) -> Should get 10 most recent (Gamma 5..1, Beta 5..1)
	resp, err := app.Client.Get(app.Server.URL + "/api/polls?page=1")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var page1 []*domain.Poll
	err = json.NewDecoder(resp.Body).Decode(&page1)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Len(t, page1, 10, "Page 1 should have 10 items")
	// Verify ordering (newest first)
	// Since we inserted Alpha, Beta, Gamma in order, Gamma is newest.
	// So page 1 should contain Gammas and Betas.
	assert.Contains(t, page1[0].Title, "Gamma", "Newest should be Gamma")

	// 2. Test Pagination (Page 2) -> Should get remaining 5 (Alpha 5..1)
	resp, err = app.Client.Get(app.Server.URL + "/api/polls?page=2")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var page2 []*domain.Poll
	err = json.NewDecoder(resp.Body).Decode(&page2)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Len(t, page2, 5, "Page 2 should have 5 items")
	assert.Contains(t, page2[0].Title, "Alpha", "Page 2 should start with Alpha")

	// 3. Test Search ("Beta") -> Should get 5 items
	resp, err = app.Client.Get(app.Server.URL + "/api/polls?q=Beta")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var searchResults []*domain.Poll
	err = json.NewDecoder(resp.Body).Decode(&searchResults)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Len(t, searchResults, 5, "Search for Beta should return 5 items")
	for _, p := range searchResults {
		assert.Contains(t, p.Title, "Beta")
	}
}

// TestListPollsSorted checks if polls are sorted by total votes descending
func TestListPollsSorted(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	app := setupTestApp(t)
	defer app.Teardown(t)

	// Create 3 polls
	// Poll A: 0 votes
	// Poll B: 10 votes
	// Poll C: 5 votes
	// Expected Order: B, C, A

	polls := make(map[string]uuid.UUID)
	titles := []string{"Poll A", "Poll B", "Poll C"}

	for _, title := range titles {
		body, _ := json.Marshal(map[string]interface{}{
			"title":       title,
			"description": "Desc",
			"options":     []string{"Opt1", "Opt2"},
		})
		resp, err := app.Client.Post(app.Server.URL+"/api/polls", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var p domain.Poll
		json.NewDecoder(resp.Body).Decode(&p)
		polls[title] = p.ID
		resp.Body.Close()
	}

	// Insert votes
	// Poll B: 10 votes (for Opt1)
	for i := 0; i < 10; i++ {
		_, err := app.DB.Exec(`INSERT INTO votes (poll_id, option_id, voter_ip) VALUES ($1, (SELECT id FROM poll_options WHERE poll_id=$1 AND text='Opt1'), $2)`, polls["Poll B"], fmt.Sprintf("1.1.1.%d", i))
		require.NoError(t, err)
	}
	// Poll C: 5 votes (for Opt1)
	for i := 0; i < 5; i++ {
		_, err := app.DB.Exec(`INSERT INTO votes (poll_id, option_id, voter_ip) VALUES ($1, (SELECT id FROM poll_options WHERE poll_id=$1 AND text='Opt1'), $2)`, polls["Poll C"], fmt.Sprintf("2.2.2.%d", i))
		require.NoError(t, err)
	}

	// Summarize
	err := app.SummarySvc.SummarizeAllVotes(context.Background())
	require.NoError(t, err)

	// List
	resp, err := app.Client.Get(app.Server.URL + "/api/polls?page=1")
	require.NoError(t, err)

	var list []*domain.Poll
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()

	require.GreaterOrEqual(t, len(list), 3)

	// We might have other polls from other tests if DB isn't cleaned (Docker container is fresh per test run usually, but let's check carefully).
	// Since setupTestApp starts a new container every time, it's fresh.

	// Check order
	assert.Equal(t, "Poll B", list[0].Title) // 10 votes
	assert.Equal(t, "Poll C", list[1].Title) // 5 votes
	assert.Equal(t, "Poll A", list[2].Title) // 0 votes
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

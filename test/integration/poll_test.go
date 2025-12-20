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
	"github.com/poll/api/internal/core/services"
)

// TestPollFlow is the main black-box integration test.
func TestPollFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// 1. Setup Database Container
	ctx := context.Background()
	dbContainer, dbURL, err := setupPostgresContainer(ctx)
	require.NoError(t, err)
	defer func() {
		if err := dbContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}()

	// 2. Connect to Database
	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err)
	defer db.Close()

	// 3. Run Migrations
	err = applyMigrations(db)
	require.NoError(t, err)

	// 4. Setup Application (Hexagonal Composition)
	repo := pollRepo.NewPollRepository(db)
	svc := services.NewPollService(repo)
	handler := pollHttp.NewPollHandler(svc)
	router := pollHttp.NewHandler(handler)

	// 5. Start Test Server
	server := httptest.NewServer(router)
	defer server.Close()

	client := server.Client()

	// --- TEST SCENARIO START ---

	// Step 1: Create a Poll
	createPayload := map[string]interface{}{
		"title":       "Integration Test Poll",
		"description": "Testing the flow",
		"options":     []string{"Option A", "Option B", "Option C"},
	}
	body, _ := json.Marshal(createPayload)

	resp, err := client.Post(server.URL+"/api/polls", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdPoll domain.Poll
	err = json.NewDecoder(resp.Body).Decode(&createdPoll)
	require.NoError(t, err)
	resp.Body.Close()

	// Assertions on Created Poll
	assert.NotEqual(t, uuid.Nil, createdPoll.ID)
	assert.Equal(t, createPayload["title"], createdPoll.Title)
	assert.Len(t, createdPoll.Options, 3)

	// Step 2: Get the Poll by ID
	resp, err = client.Get(fmt.Sprintf("%s/api/polls/%s", server.URL, createdPoll.ID))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var fetchedPoll domain.Poll
	err = json.NewDecoder(resp.Body).Decode(&fetchedPoll)
	require.NoError(t, err)
	resp.Body.Close()

	// Step 3: Verify Contract Consistency
	assert.Equal(t, createdPoll.ID, fetchedPoll.ID)
	assert.Equal(t, createdPoll.Title, fetchedPoll.Title)
	assert.Equal(t, createdPoll.Description, fetchedPoll.Description)
	assert.Len(t, fetchedPoll.Options, 3)

	// Verify options order and content (assuming DB preserves insertion order or logic sorts them,
	// though standard SQL doesn't guarantee order without ORDER BY.
	// Our app doesn't seem to enforce order, so checking existence is safer,
	// but usually simple inserts return in order).
	// For strictness, let's map them by Text.
	createdOptions := make(map[string]uuid.UUID)
	for _, o := range createdPoll.Options {
		createdOptions[o.Text] = o.ID
	}

	for _, o := range fetchedPoll.Options {
		id, exists := createdOptions[o.Text]
		assert.True(t, exists, "Option %s should exist", o.Text)
		assert.Equal(t, id, o.ID, "Option ID should match")
	}
}

// Helper to start Postgres container
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

// Helper to apply migrations from the project file
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

		// Convention: look for files ending in "_up.sql" or just "up.sql"
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

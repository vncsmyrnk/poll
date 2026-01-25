package integration

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	handler "github.com/vncsmyrnk/poll/internal/adapters/handler/http"
	repo "github.com/vncsmyrnk/poll/internal/adapters/repository/postgres"
	"github.com/vncsmyrnk/poll/internal/core/ports"
	"github.com/vncsmyrnk/poll/internal/core/services"
)

// MockVerifier for testing
type MockVerifier struct {
	email string
	name  string
}

func (v *MockVerifier) Verify(ctx context.Context, token string, clientID string) (*ports.TokenPayload, error) {
	if token == "valid_token" {
		return &ports.TokenPayload{Email: v.email, Name: v.name}, nil
	}
	return nil, assert.AnError
}

type TestApp struct {
	DB          *sql.DB
	Server      *httptest.Server
	Client      *http.Client
	SummarySvc  ports.SummaryService
	DBContainer testcontainers.Container
}

func setupTestApp(t *testing.T) *TestApp {
	os.Setenv("JWT_SECRET", "test-secret")
	ctx := context.Background()
	dbContainer, dbURL, err := setupPostgresContainer(ctx)
	require.NoError(t, err)

	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err)

	err = applyMigrations(db)
	require.NoError(t, err)

	pollRepo := repo.NewPollRepository(db)
	voteRepo := repo.NewVoteRepository(db)
	resultRepo := repo.NewPollResultRepository(db)
	userRepo := repo.NewUserRepository(db)
	authRepo := repo.NewAuthRepository(db)
	mockVerifier := &MockVerifier{email: "test@example.com", name: "Test user"}

	svc := services.NewPollService(pollRepo, resultRepo, voteRepo)
	voteSvc := services.NewVoteService(pollRepo, voteRepo)
	userSvc := services.NewUserService(userRepo)
	authSvc := services.NewAuthService(userRepo, authRepo, mockVerifier)
	summarySvc := services.NewSummaryService(pollRepo, resultRepo)

	pollHandler := handler.NewPollHandler(svc)
	voteHandler := handler.NewVoteHandler(voteSvc)
	userhandler := handler.NewUserHandler(userSvc)
	authHandler := handler.NewAuthHandler(authSvc, "https://example.com/redirect", "", http.SameSiteLaxMode)
	router := handler.NewHandler(pollHandler, voteHandler, authHandler, userhandler, []string{"*"})

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

func createUserAndToken(t *testing.T, db *sql.DB) string {
	t.Helper()

	userID := uuid.New()
	email := fmt.Sprintf("user-%s@example.com", userID)
	name := fmt.Sprintf("User %s", userID)
	_, err := db.Exec("INSERT INTO users (id, email, name) VALUES ($1, $2, $3)", userID, email, name)
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"sub":   userID.String(),
		"email": email,
		"exp":   time.Now().Add(15 * time.Minute).Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte("test-secret"))
	require.NoError(t, err)
	return signedToken
}

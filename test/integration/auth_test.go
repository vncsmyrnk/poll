package integration

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	handler "github.com/poll/api/internal/adapters/handler/http"
	repo "github.com/poll/api/internal/adapters/repository/postgres"
	"github.com/poll/api/internal/core/ports"
	"github.com/poll/api/internal/core/services"
)

// MockVerifier for testing
type MockVerifier struct {
	email string
}

func (v *MockVerifier) Verify(ctx context.Context, token string, clientID string) (*ports.TokenPayload, error) {
	if token == "valid_token" {
		return &ports.TokenPayload{Email: v.email}, nil
	}
	return nil, assert.AnError
}

func setupAuthTestApp(t *testing.T) *TestApp {
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
	authRepo := repo.NewAuthRepository(db)
	userRepo := repo.NewUserRepository(db)

	svc := services.NewPollService(pollRepo, resultRepo)
	voteSvc := services.NewVoteService(pollRepo, voteRepo)
	summarySvc := services.NewSummaryService(pollRepo, resultRepo)

	mockVerifier := &MockVerifier{email: "test@example.com"}
	authSvc := services.NewAuthService(userRepo, authRepo, mockVerifier)

	pollHandler := handler.NewPollHandler(svc)
	voteHandler := handler.NewVoteHandler(voteSvc)
	authHandler := handler.NewAuthHandler(authSvc, "https://example.com/redirect")
	router := handler.NewHandler(pollHandler, voteHandler, authHandler, []string{"*"})

	server := httptest.NewServer(router)

	return &TestApp{
		DB:          db,
		Server:      server,
		Client:      server.Client(),
		SummarySvc:  summarySvc,
		DBContainer: dbContainer,
	}
}

func TestAuthFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	app := setupAuthTestApp(t)
	defer app.Teardown(t)

	// 1. Callback with Valid Credential
	form := url.Values{}
	form.Add("credential", "valid_token")

	// Configure client to NOT follow redirects to check cookies and location
	app.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := app.Client.PostForm(app.Server.URL+"/oauth/callback", form)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusSeeOther, resp.StatusCode)

	location, err := resp.Location()
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/redirect", location.String())

	// Check Cookies
	var accessToken, refreshToken string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "access_token" {
			accessToken = cookie.Value
		}
		if cookie.Name == "refresh_token" {
			refreshToken = cookie.Value
		}
	}

	assert.NotEmpty(t, accessToken, "access_token cookie should be set")
	assert.NotEmpty(t, refreshToken, "refresh_token cookie should be set")

	// 2. Refresh Token
	// Wait a bit just to be sure
	time.Sleep(1200 * time.Millisecond)

	req, err := http.NewRequest("POST", app.Server.URL+"/oauth/refresh", nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})

	resp, err = app.Client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Should get a new access token
	newAccessToken := ""
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "access_token" {
			newAccessToken = cookie.Value
		}
	}
	assert.NotEmpty(t, newAccessToken, "new access_token should be returned")
	assert.NotEqual(t, accessToken, newAccessToken, "access token should be different (rotated/new)")
}

func TestAuthFlow_Invalid(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	app := setupAuthTestApp(t)
	defer app.Teardown(t)

	// Invalid Credential
	form := url.Values{}
	form.Add("credential", "bad_token")

	resp, err := app.Client.PostForm(app.Server.URL+"/oauth/callback", form)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Invalid Refresh Token
	req, err := http.NewRequest("POST", app.Server.URL+"/oauth/refresh", nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "garbage"})

	resp, err = app.Client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMe(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	app := setupTestApp(t)
	defer app.Teardown(t)

	token := createUserAndToken(t, app.DB)

	req, err := http.NewRequest("GET", app.Server.URL+"/api/me", nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Client.Do(req)
	require.NoError(t, err)
	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		t.Logf("Response Body: %s", buf.String())
	}
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var user map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&user)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Extract User ID from token to fetch from DB
	claims := jwt.MapClaims{}
	_, _, err = new(jwt.Parser).ParseUnverified(token, claims)
	require.NoError(t, err)

	sub, ok := claims["sub"].(string)
	require.True(t, ok)

	// Fetch expected user from DB
	var dbID, dbEmail, dbName string
	err = app.DB.QueryRow("SELECT id, email, name FROM users WHERE id = $1", sub).Scan(&dbID, &dbEmail, &dbName)
	require.NoError(t, err)

	assert.Equal(t, dbID, user["id"])
	assert.Equal(t, dbEmail, user["email"])
	assert.Equal(t, dbName, user["name"])
}

func TestGetMe_Unauthorized(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	app := setupTestApp(t)
	defer app.Teardown(t)

	// Call /api/me without token
	req, err := http.NewRequest("GET", app.Server.URL+"/api/me", nil)
	require.NoError(t, err)

	resp, err := app.Client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

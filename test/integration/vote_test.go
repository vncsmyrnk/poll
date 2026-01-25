package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vncsmyrnk/poll/internal/core/domain"
)

func TestGetMyVote(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	app := setupTestApp(t)
	defer app.Teardown(t)

	// 1. Create Poll
	createPayload := map[string]interface{}{
		"title":       "My Vote Test",
		"description": "Testing my-vote endpoint",
		"options":     []string{"Yes", "No"},
	}
	body, _ := json.Marshal(createPayload)
	resp, err := app.Client.Post(app.Server.URL+"/api/polls", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	var poll domain.Poll
	json.NewDecoder(resp.Body).Decode(&poll)
	resp.Body.Close()

	// 2. Check My Vote (Before Voting) -> Should be 404
	token := createUserAndToken(t, app.DB)
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/polls/%s/my-vote", app.Server.URL, poll.ID), nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	resp, err = app.Client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	// 3. Vote
	voteBody, _ := json.Marshal(map[string]interface{}{"option_id": poll.Options[0].ID})
	req, err = http.NewRequest("POST", fmt.Sprintf("%s/api/polls/%s/votes", app.Server.URL, poll.ID), bytes.NewReader(voteBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	resp, err = app.Client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// 4. Check My Vote (After Voting) -> Should be 200 and contain option_id
	req, err = http.NewRequest("GET", fmt.Sprintf("%s/api/polls/%s/my-vote", app.Server.URL, poll.ID), nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	resp, err = app.Client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var myVote map[string]string
	err = json.NewDecoder(resp.Body).Decode(&myVote)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, poll.Options[0].ID.String(), myVote["option_id"])
}

func TestVoteSwitching(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	app := setupTestApp(t)
	defer app.Teardown(t)

	// 1. Create Poll
	createPayload := map[string]interface{}{
		"title":       "Vote Switch Test",
		"description": "Testing vote switching",
		"options":     []string{"Opt A", "Opt B"},
	}
	body, _ := json.Marshal(createPayload)
	resp, err := app.Client.Post(app.Server.URL+"/api/polls", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	var poll domain.Poll
	json.NewDecoder(resp.Body).Decode(&poll)
	resp.Body.Close()

	// 2. Vote for Option A
	token := createUserAndToken(t, app.DB)
	voteBodyA, _ := json.Marshal(map[string]interface{}{"option_id": poll.Options[0].ID})

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/polls/%s/votes", app.Server.URL, poll.ID), bytes.NewReader(voteBodyA))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	resp, err = app.Client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Verify Vote for A exists
	var countA int
	err = app.DB.QueryRow("SELECT COUNT(*) FROM votes WHERE poll_id=$1 AND option_id=$2 AND deleted_at IS NULL", poll.ID, poll.Options[0].ID).Scan(&countA)
	require.NoError(t, err)
	assert.Equal(t, 1, countA)

	// 3. Vote for Option A AGAIN (Should fail - same option)
	req, err = http.NewRequest("POST", fmt.Sprintf("%s/api/polls/%s/votes", app.Server.URL, poll.ID), bytes.NewReader(voteBodyA))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	resp, err = app.Client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, resp.StatusCode)

	// 4. Vote for Option B (Should succeed and switch vote)
	voteBodyB, _ := json.Marshal(map[string]interface{}{"option_id": poll.Options[1].ID})
	req, err = http.NewRequest("POST", fmt.Sprintf("%s/api/polls/%s/votes", app.Server.URL, poll.ID), bytes.NewReader(voteBodyB))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	resp, err = app.Client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Verify Vote for A is gone (soft deleted)
	err = app.DB.QueryRow("SELECT COUNT(*) FROM votes WHERE poll_id=$1 AND option_id=$2 AND deleted_at IS NULL", poll.ID, poll.Options[0].ID).Scan(&countA)
	require.NoError(t, err)
	assert.Equal(t, 0, countA)

	// Verify Vote for B exists
	var countB int
	err = app.DB.QueryRow("SELECT COUNT(*) FROM votes WHERE poll_id=$1 AND option_id=$2 AND deleted_at IS NULL", poll.ID, poll.Options[1].ID).Scan(&countB)
	require.NoError(t, err)
	assert.Equal(t, 1, countB)
}

package http

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vncsmyrnk/poll/internal/core/domain"
	"github.com/vncsmyrnk/poll/internal/core/ports"
)

type VoteHandler struct {
	service ports.VoteService
}

func NewVoteHandler(service ports.VoteService) *VoteHandler {
	return &VoteHandler{
		service: service,
	}
}

type voteRequest struct {
	OptionID uuid.UUID `json:"option_id"`
}

// VoteOnPoll godoc
// @Summary      Casts a vote on a poll
// @Tags         polls
// @Accept       json
// @Param        Authorization  header  string       true  "authorization header"
// @Param        request        body    voteRequest  true  "request body"
// @Param        id             path    int          true  "poll id"
// @Success      201
// @Failure      400
// @Failure      404
// @Failure      409
// @Failure      500
// @Router       /polls/{id}/votes [post]
func (h *VoteHandler) VoteOnPoll(w http.ResponseWriter, r *http.Request) {
	pollIDStr := chi.URLParam(r, "id")
	pollID, err := uuid.Parse(pollIDStr)
	if err != nil {
		http.Error(w, "invalid poll id", http.StatusBadRequest)
		return
	}

	var req voteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}

	userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized: missing user context", http.StatusUnauthorized)
		return
	}

	input := ports.VoteInput{
		PollID:   pollID,
		OptionID: req.OptionID,
		UserID:   userID,
		VoterIP:  ip,
	}

	if err := h.service.Vote(r.Context(), input); err != nil {
		if errors.Is(err, domain.ErrInvalidOption) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, domain.ErrAlreadyVoted) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, domain.ErrPollNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// GetMyVote godoc
// @Summary      Gets the user vote on a poll
// @Tags         polls
// @Produce      json
// @Param        Authorization  header  string       true  "authorization header"
// @Param        id             path    int          true  "poll id"
// @Success      200  {object}  domain.Vote
// @Failure      400
// @Failure      404
// @Failure      500
// @Router       /polls/{id}/my-vote [get]
func (h *VoteHandler) GetMyVote(w http.ResponseWriter, r *http.Request) {
	pollIDStr := chi.URLParam(r, "id")
	pollID, err := uuid.Parse(pollIDStr)
	if err != nil {
		http.Error(w, "invalid poll id", http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized: missing user context", http.StatusUnauthorized)
		return
	}

	vote, err := h.service.GetUserVote(r.Context(), pollID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotVoted) {
			http.Error(w, "vote not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]uuid.UUID{
		"option_id": vote.OptionID,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

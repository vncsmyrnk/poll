package http

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/poll/api/internal/core/domain"
	"github.com/poll/api/internal/core/ports"
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

func (h *VoteHandler) Unvote(w http.ResponseWriter, r *http.Request) {
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

	if err := h.service.Unvote(r.Context(), pollID, userID); err != nil {
		if errors.Is(err, domain.ErrDidNotVoted) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

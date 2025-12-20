package http

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

	input := ports.VoteInput{
		PollID:   pollID,
		OptionID: req.OptionID,
		VoterIP:  ip,
	}

	if err := h.service.Vote(r.Context(), input); err != nil {
		if err.Error() == "invalid option for this poll" {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err.Error() == "user has already voted" {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/poll/api/internal/core/ports"
)

type PollHandler struct {
	service ports.PollService
}

func NewPollHandler(service ports.PollService) *PollHandler {
	return &PollHandler{
		service: service,
	}
}

type createPollRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Options     []string `json:"options"`
}

func (h *PollHandler) CreatePoll(w http.ResponseWriter, r *http.Request) {
	var req createPollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input := ports.CreatePollInput{
		Title:       req.Title,
		Description: req.Description,
		Options:     req.Options,
	}

	poll, err := h.service.Create(r.Context(), input)
	if err != nil {
		// In a real app, check error type for validation vs internal error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(poll); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (h *PollHandler) GetPoll(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing poll id", http.StatusBadRequest)
		return
	}

	poll, err := h.service.GetPoll(r.Context(), id)
	if err != nil {
		if err.Error() == "invalid poll id" {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.Contains(err.Error(), "poll not found") {
			http.Error(w, "poll not found", http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(poll); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

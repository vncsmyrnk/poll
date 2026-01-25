package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/vncsmyrnk/poll/internal/core/ports"
)

type UserHandler struct {
	service ports.UserService
}

func NewUserHandler(service ports.UserService) *UserHandler {
	return &UserHandler{
		service: service,
	}
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized: missing user context", http.StatusUnauthorized)
		return
	}

	user, err := h.service.GetByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to fetch user: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

package domain

import (
	"time"

	"github.com/google/uuid"
)

type Poll struct {
	ID          uuid.UUID    `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Options     []PollOption `json:"options"`
	CreatedAt   time.Time    `json:"created_at"`
	ExpiresAt   *time.Time   `json:"expires_at,omitempty"`
}

type PollOption struct {
	ID        uuid.UUID `json:"id"`
	PollID    uuid.UUID `json:"poll_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type PollOptionStats struct {
	VoteCount  int64
	Percentage float64
}

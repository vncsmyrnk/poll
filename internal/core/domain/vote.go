package domain

import (
	"time"

	"github.com/google/uuid"
)

type Vote struct {
	ID        uuid.UUID `json:"id"`
	PollID    uuid.UUID `json:"poll_id"`
	OptionID  uuid.UUID `json:"option_id"`
	VoterIP   string    `json:"voter_ip"`
	CreatedAt time.Time `json:"created_at"`
}

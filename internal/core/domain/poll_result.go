package domain

import (
	"time"

	"github.com/google/uuid"
)

type PollResult struct {
	PollID        uuid.UUID
	OptionID      uuid.UUID
	VoteCount     int64
	LastUpdatedAt time.Time
}

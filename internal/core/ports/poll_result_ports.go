package ports

import (
	"context"

	"github.com/google/uuid"
)

type PollResultRepository interface {
	GetOptionStats(ctx context.Context, pollID uuid.UUID, optionID uuid.UUID) (int64, float64, error)
}

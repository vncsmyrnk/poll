package ports

import (
	"context"

	"github.com/google/uuid"
)

type PollResultRepository interface {
	SummarizeVotes(ctx context.Context, pollID uuid.UUID) error
	GetOptionStats(ctx context.Context, pollID uuid.UUID, optionID uuid.UUID) (int64, float64, error)
}

type SummaryService interface {
	SummarizeAllVotes(ctx context.Context) error
}

package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/poll/api/internal/core/domain"
)

type PollResultRepository interface {
	SummarizeVotes(ctx context.Context, pollID uuid.UUID) error
	GetPollOptionStats(ctx context.Context, pollID uuid.UUID) (map[uuid.UUID]domain.PollOptionStats, error)
}

type SummaryService interface {
	SummarizeAllVotes(ctx context.Context) error
}

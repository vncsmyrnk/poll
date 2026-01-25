package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/vncsmyrnk/poll/internal/core/domain"
)

type PollResultRepository interface {
	ProcessVotes(ctx context.Context, pollID uuid.UUID) error
	GetPollsWithUnprocessedVotes(ctx context.Context) ([]uuid.UUID, error)
	GetPollOptionStats(ctx context.Context, pollID uuid.UUID) (map[uuid.UUID]domain.PollOptionStats, error)
}

type SummaryService interface {
	SummarizeAllVotes(ctx context.Context) error
}

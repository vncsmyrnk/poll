package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/poll/api/internal/core/domain"
)

type VoteRepository interface {
	SaveVote(ctx context.Context, vote *domain.Vote) error
	HasVoted(ctx context.Context, pollID uuid.UUID, voterIP string) (bool, error)
}

type VoteInput struct {
	PollID   uuid.UUID
	OptionID uuid.UUID
	VoterIP  string
}

type VoteService interface {
	Vote(ctx context.Context, input VoteInput) error
}

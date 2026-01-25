package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/vncsmyrnk/poll/internal/core/domain"
)

type VoteRepository interface {
	SaveVote(ctx context.Context, vote *domain.Vote) error
	DeleteVote(ctx context.Context, pollID, userID uuid.UUID) error
	HasVoted(ctx context.Context, pollID, userID uuid.UUID) (bool, error)
	HasVotedOnOption(ctx context.Context, optionID, userID uuid.UUID) (bool, error)
	GetVote(ctx context.Context, pollID, userID uuid.UUID) (*domain.Vote, error)
}

type VoteInput struct {
	PollID   uuid.UUID
	OptionID uuid.UUID
	UserID   uuid.UUID
	VoterIP  string
}

type VoteService interface {
	Vote(ctx context.Context, input VoteInput) error
	GetUserVote(ctx context.Context, pollID, userID uuid.UUID) (*domain.Vote, error)
}

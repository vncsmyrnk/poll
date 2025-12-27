package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/poll/api/internal/core/domain"
	"github.com/poll/api/internal/core/ports"
)

type voteService struct {
	pollRepo ports.PollRepository
	voteRepo ports.VoteRepository
}

func NewVoteService(pollRepo ports.PollRepository, voteRepo ports.VoteRepository) ports.VoteService {
	return &voteService{
		pollRepo: pollRepo,
		voteRepo: voteRepo,
	}
}

func (s *voteService) Vote(ctx context.Context, input ports.VoteInput) error {
	poll, err := s.pollRepo.GetByID(ctx, input.PollID)
	if err != nil {
		return err
	}

	validOption := false
	for _, opt := range poll.Options {
		if opt.ID == input.OptionID {
			validOption = true
			break
		}
	}
	if !validOption {
		return domain.ErrInvalidOption
	}

	hasVoted, err := s.voteRepo.HasVoted(ctx, input.PollID, input.UserID)
	if err != nil {
		return err
	}
	if hasVoted {
		return domain.ErrAlreadyVoted
	}

	vote := &domain.Vote{
		ID:        uuid.New(),
		PollID:    input.PollID,
		OptionID:  input.OptionID,
		UserID:    input.UserID,
		VoterIP:   input.VoterIP,
		CreatedAt: time.Now(),
	}

	return s.voteRepo.SaveVote(ctx, vote)
}

func (s *voteService) Unvote(ctx context.Context, pollID, userID uuid.UUID) error {
	hasVoted, err := s.voteRepo.HasVoted(ctx, pollID, userID)
	if err != nil {
		return err
	}

	if !hasVoted {
		return domain.ErrUserNotVoted
	}

	return s.voteRepo.DeleteVote(ctx, pollID, userID)
}

package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/poll/api/internal/core/domain"
	"github.com/poll/api/internal/core/ports"
)

type pollService struct {
	pollRepo       ports.PollRepository
	pollResultRepo ports.PollResultRepository
	voteRepo       ports.VoteRepository
}

func NewPollService(pollRepo ports.PollRepository, pollResultRepo ports.PollResultRepository, voteRepo ports.VoteRepository) ports.PollService {
	return &pollService{
		pollRepo:       pollRepo,
		pollResultRepo: pollResultRepo,
		voteRepo:       voteRepo,
	}
}

func (s *pollService) Create(ctx context.Context, input ports.CreatePollInput) (*domain.Poll, error) {
	if input.Title == "" {
		return nil, errors.New("title is required")
	}
	if len(input.Options) < 2 {
		return nil, errors.New("at least two options are required")
	}

	pollID := uuid.New()
	now := time.Now()

	poll := &domain.Poll{
		ID:          pollID,
		Title:       input.Title,
		Description: input.Description,
		CreatedAt:   now,
	}

	for _, optText := range input.Options {
		if optText == "" {
			continue
		}
		poll.Options = append(poll.Options, domain.PollOption{
			ID:        uuid.New(),
			PollID:    pollID,
			Text:      optText,
			CreatedAt: now,
		})
	}

	if len(poll.Options) < 2 {
		return nil, errors.New("at least two valid options are required")
	}

	err := s.pollRepo.Save(ctx, poll)
	if err != nil {
		return nil, err
	}

	return poll, nil
}

func (s *pollService) GetPoll(ctx context.Context, id string) (*domain.Poll, error) {
	pollID, err := uuid.Parse(id)
	if err != nil {
		return nil, domain.ErrInvalidPollID
	}

	poll, err := s.pollRepo.GetByID(ctx, pollID)
	if err != nil {
		return nil, err
	}

	return poll, nil
}

func (s *pollService) ListPolls(ctx context.Context, input ports.ListPollsInput) ([]*domain.Poll, error) {
	const pageSize = 10
	const maxItems = 100

	page := input.Page
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * pageSize

	if offset >= maxItems {
		return []*domain.Poll{}, nil
	}

	var polls []*domain.Poll
	var err error

	if input.Query == "" {
		polls, err = s.pollRepo.List(ctx, pageSize, offset)
	} else {
		polls, err = s.pollRepo.Search(ctx, pageSize, offset, input.Query)
	}

	if err != nil {
		return nil, err
	}

	return polls, nil
}

func (s *pollService) GetPollStats(ctx context.Context, id string, userID uuid.UUID) (map[uuid.UUID]domain.PollOptionStats, error) {
	pollID, err := uuid.Parse(id)
	if err != nil {
		return nil, domain.ErrInvalidPollID
	}

	_, err = s.pollRepo.GetByID(ctx, pollID)
	if err != nil {
		return nil, err
	}

	hasVoted, err := s.voteRepo.HasVoted(ctx, pollID, userID)
	if err != nil {
		return nil, err
	}

	if !hasVoted {
		return nil, domain.ErrUserNotVoted
	}

	return s.pollResultRepo.GetPollOptionStats(ctx, pollID)
}

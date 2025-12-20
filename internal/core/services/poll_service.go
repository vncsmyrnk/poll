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
	repo ports.PollRepository
}

func NewPollService(repo ports.PollRepository) ports.PollService {
	return &pollService{
		repo: repo,
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

	err := s.repo.Save(ctx, poll)
	if err != nil {
		return nil, err
	}

	return poll, nil
}

func (s *pollService) GetPoll(ctx context.Context, id string) (*domain.Poll, error) {
	pollID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid poll id")
	}

	return s.repo.GetByID(ctx, pollID)
}

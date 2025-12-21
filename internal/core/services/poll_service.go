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
}

func NewPollService(pollRepo ports.PollRepository, pollResultRepo ports.PollResultRepository) ports.PollService {
	return &pollService{
		pollRepo:       pollRepo,
		pollResultRepo: pollResultRepo,
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

	// Fetch option stats in parallel
	type optionStat struct {
		Index      int
		Count      int64
		Percentage float64
		Err        error
	}

	statChan := make(chan optionStat, len(poll.Options))

	for i, opt := range poll.Options {
		go func(index int, pID, optID uuid.UUID) {
			count, percentage, err := s.pollResultRepo.GetOptionStats(ctx, pID, optID)
			statChan <- optionStat{Index: index, Count: count, Percentage: percentage, Err: err}
		}(i, poll.ID, opt.ID)
	}

	for i := 0; i < len(poll.Options); i++ {
		stat := <-statChan
		if stat.Err != nil {
			return nil, stat.Err
		}
		poll.Options[stat.Index].VoteCount = stat.Count
		poll.Options[stat.Index].Percentage = stat.Percentage
	}
	close(statChan)

	return poll, nil
}

package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/poll/api/internal/core/ports"
)

type summaryService struct {
	pollRepo       ports.PollRepository
	pollResultRepo ports.PollResultRepository
}

func NewSummaryService(pollRepo ports.PollRepository, pollResultRepo ports.PollResultRepository) ports.SummaryService {
	return &summaryService{
		pollRepo:       pollRepo,
		pollResultRepo: pollResultRepo,
	}
}

func (s *summaryService) SummarizeAllVotes(ctx context.Context) error {
	polls, err := s.pollRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch all polls: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(polls))

	for _, poll := range polls {
		wg.Add(1)
		go func(pID [16]byte) { // passing ID by value (uuid.UUID is [16]byte) to avoid closure issues
			defer wg.Done()
			if err := s.pollResultRepo.SummarizeVotes(ctx, pID); err != nil {
				errChan <- fmt.Errorf("failed to summarize poll %s: %w", pID, err)
			}
		}(poll.ID)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

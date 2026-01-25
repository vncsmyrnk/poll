package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/vncsmyrnk/poll/internal/core/ports"
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
	pollIDs, err := s.pollResultRepo.GetPollsWithUnprocessedVotes(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch polls with unprocessed votes: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(pollIDs))

	for _, pID := range pollIDs {
		wg.Add(1)
		go func(id uuid.UUID) {
			defer wg.Done()
			if err := s.pollResultRepo.ProcessVotes(ctx, id); err != nil {
				errChan <- fmt.Errorf("failed to summarize poll %s: %w", id, err)
			}
		}(pID)
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

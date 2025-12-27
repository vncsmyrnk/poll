package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/poll/api/internal/core/domain"
	"github.com/poll/api/internal/core/ports"
)

type pollResultRepository struct {
	db *sql.DB
}

func NewPollResultRepository(db *sql.DB) ports.PollResultRepository {
	return &pollResultRepository{
		db: db,
	}
}

func (r *pollResultRepository) GetPollOptionStats(ctx context.Context, pollID uuid.UUID) (map[uuid.UUID]domain.PollOptionStats, error) {
	query := `
		SELECT poll_id, option_id, vote_count
		FROM poll_results
		WHERE poll_id = $1
	`

	rows, err := r.db.QueryContext(ctx, query, pollID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats batch: %w", err)
	}
	defer rows.Close()

	type rawStat struct {
		PollID   uuid.UUID
		OptionID uuid.UUID
		Count    int64
	}
	var rawStats []rawStat
	pollTotals := make(map[uuid.UUID]int64)

	for rows.Next() {
		var s rawStat
		if err := rows.Scan(&s.PollID, &s.OptionID, &s.Count); err != nil {
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}
		rawStats = append(rawStats, s)
		pollTotals[s.PollID] += s.Count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating stats: %w", err)
	}

	result := make(map[uuid.UUID]domain.PollOptionStats)
	for _, s := range rawStats {
		total := pollTotals[s.PollID]
		percentage := 0.0
		if total > 0 {
			percentage = (float64(s.Count) / float64(total)) * 100
		}

		result[s.OptionID] = domain.PollOptionStats{
			VoteCount:  s.Count,
			Percentage: percentage,
		}
	}

	return result, nil
}

func (r *pollResultRepository) SummarizeVotes(ctx context.Context, pollID uuid.UUID) error {
	query := `
		INSERT INTO poll_results (poll_id, option_id, vote_count, last_updated_at)
		SELECT poll_id, option_id, COUNT(*), NOW()
		FROM votes
		WHERE poll_id = $1
		GROUP BY poll_id, option_id
		ON CONFLICT (poll_id, option_id) DO UPDATE
		SET vote_count = EXCLUDED.vote_count,
		    last_updated_at = NOW();
	`

	_, err := r.db.ExecContext(ctx, query, pollID)
	if err != nil {
		return fmt.Errorf("failed to summarize votes for poll %s: %w", pollID, err)
	}

	return nil
}

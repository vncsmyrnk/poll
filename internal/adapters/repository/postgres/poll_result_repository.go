package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/vncsmyrnk/poll/internal/core/domain"
	"github.com/vncsmyrnk/poll/internal/core/ports"
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

func (r *pollResultRepository) ProcessVotes(ctx context.Context, pollID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	queryInc := `
		WITH processed_votes AS (
			UPDATE votes
			SET status = 'valid'
			WHERE poll_id = $1 AND status = 'pending' AND deleted_at IS NULL
			RETURNING option_id
		)
		INSERT INTO poll_results (poll_id, option_id, vote_count, last_updated_at)
		SELECT $1, option_id, COUNT(*), NOW()
		FROM processed_votes
		GROUP BY option_id
		ON CONFLICT (poll_id, option_id) DO UPDATE
		SET vote_count = poll_results.vote_count + EXCLUDED.vote_count,
		    last_updated_at = NOW();
	`
	_, err = tx.ExecContext(ctx, queryInc, pollID)
	if err != nil {
		return fmt.Errorf("failed to process increments for poll %s: %w", pollID, err)
	}

	queryDec := `
		WITH deleted_votes AS (
			UPDATE votes
			SET status = 'invalid'
			WHERE poll_id = $1 AND status = 'valid' AND deleted_at IS NOT NULL
			RETURNING option_id
		)
		INSERT INTO poll_results (poll_id, option_id, vote_count, last_updated_at)
		SELECT $1, option_id, -COUNT(*), NOW()
		FROM deleted_votes
		GROUP BY option_id
		ON CONFLICT (poll_id, option_id) DO UPDATE
		SET vote_count = poll_results.vote_count + EXCLUDED.vote_count,
		    last_updated_at = NOW();
	`
	_, err = tx.ExecContext(ctx, queryDec, pollID)
	if err != nil {
		return fmt.Errorf("failed to process decrements for poll %s: %w", pollID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *pollResultRepository) GetPollsWithUnprocessedVotes(ctx context.Context) ([]uuid.UUID, error) {
	query := `
		SELECT DISTINCT poll_id
		FROM votes
		WHERE (status = 'pending' AND deleted_at IS NULL)
		   OR (status = 'valid' AND deleted_at IS NOT NULL)
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get polls with unprocessed votes: %w", err)
	}
	defer rows.Close()

	var pollIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan poll id: %w", err)
		}
		pollIDs = append(pollIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating poll ids: %w", err)
	}

	return pollIDs, nil
}

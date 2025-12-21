package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
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

func (r *pollResultRepository) GetOptionStats(ctx context.Context, pollID uuid.UUID, optionID uuid.UUID) (int64, float64, error) {
	query := `
		WITH total_votes AS (
			SELECT SUM(vote_count) as total
			FROM poll_results
			WHERE poll_id = $1
		)
		SELECT
			COALESCE(vote_count, 0),
			CASE
				WHEN (SELECT total FROM total_votes) > 0
				THEN (COALESCE(vote_count, 0)::float / (SELECT total FROM total_votes)) * 100
				ELSE 0
			END
		FROM poll_results
		WHERE poll_id = $1 AND option_id = $2
	`

	var count int64
	var percentage float64

	err := r.db.QueryRowContext(ctx, query, pollID, optionID).Scan(&count, &percentage)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, 0.0, nil
		}
		return 0, 0.0, fmt.Errorf("failed to get option stats: %w", err)
	}

	return count, percentage, nil
}

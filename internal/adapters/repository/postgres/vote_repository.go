package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/poll/api/internal/core/domain"
	"github.com/poll/api/internal/core/ports"
)

type voteRepository struct {
	db *sql.DB
}

func NewVoteRepository(db *sql.DB) ports.VoteRepository {
	return &voteRepository{
		db: db,
	}
}

func (r *voteRepository) SaveVote(ctx context.Context, vote *domain.Vote) error {
	query := `
		INSERT INTO votes (id, poll_id, option_id, user_id, voter_ip)
		VALUES ($1, $2, $3, $4, $5);
	`
	_, err := r.db.ExecContext(ctx, query, vote.ID, vote.PollID, vote.OptionID, vote.UserID, vote.VoterIP)
	if err != nil {
		return fmt.Errorf("failed to save vote: %w", err)
	}
	return nil
}

func (r *voteRepository) DeleteVote(ctx context.Context, pollID, userID uuid.UUID) error {
	query := `
		UPDATE votes SET deleted_at = NOW() WHERE poll_id = $1 and user_id = $2
	`
	_, err := r.db.ExecContext(ctx, query, pollID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete vote: %w", err)
	}
	return nil
}

func (r *voteRepository) HasVoted(ctx context.Context, pollID uuid.UUID, userID uuid.UUID) (bool, error) {
	query := `SELECT 1 FROM votes WHERE poll_id = $1 AND user_id = $2 AND deleted_at IS NULL LIMIT 1`
	var exists int
	err := r.db.QueryRowContext(ctx, query, pollID, userID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existing vote: %w", err)
	}
	return true, nil
}

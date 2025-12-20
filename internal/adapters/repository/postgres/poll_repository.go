package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/poll/api/internal/core/domain"
	"github.com/poll/api/internal/core/ports"
)

type pollRepository struct {
	db *sql.DB
}

func NewPollRepository(db *sql.DB) ports.PollRepository {
	return &pollRepository{
		db: db,
	}
}

func (r *pollRepository) Save(ctx context.Context, poll *domain.Poll) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	queryPoll := `
		INSERT INTO polls (id, title, description, expires_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err = tx.ExecContext(ctx, queryPoll, poll.ID, poll.Title, poll.Description, poll.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to insert poll: %w", err)
	}

	queryOption := `
		INSERT INTO poll_options (id, poll_id, text)
		VALUES ($1, $2, $3)
	`
	stmt, err := tx.PrepareContext(ctx, queryOption)
	if err != nil {
		return fmt.Errorf("failed to prepare option statement: %w", err)
	}
	defer stmt.Close()

	for _, opt := range poll.Options {
		_, err = stmt.ExecContext(ctx, opt.ID, opt.PollID, opt.Text)
		if err != nil {
			return fmt.Errorf("failed to insert option: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *pollRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Poll, error) {
	queryPoll := `
		SELECT id, title, description, created_at, expires_at
		FROM polls
		WHERE id = $1
	`

	var poll domain.Poll
	err := r.db.QueryRowContext(ctx, queryPoll, id).Scan(
		&poll.ID, &poll.Title, &poll.Description, &poll.CreatedAt, &poll.ExpiresAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrPollNotFound
		}
		return nil, fmt.Errorf("failed to get poll: %w", err)
	}

	queryOptions := `
		SELECT id, poll_id, text, created_at
		FROM poll_options
		WHERE poll_id = $1
	`
	rows, err := r.db.QueryContext(ctx, queryOptions, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get poll options: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var opt domain.PollOption
		if err := rows.Scan(&opt.ID, &opt.PollID, &opt.Text, &opt.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan option: %w", err)
		}
		poll.Options = append(poll.Options, opt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating options: %w", err)
	}

	return &poll, nil
}

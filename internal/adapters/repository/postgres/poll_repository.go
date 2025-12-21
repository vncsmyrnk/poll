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

	options, err := r.fetchOptions(ctx, poll.ID)
	if err != nil {
		return nil, err
	}
	poll.Options = options

	return &poll, nil
}

func (r *pollRepository) GetAll(ctx context.Context) ([]*domain.Poll, error) {
	query := `
		SELECT id, title, description, created_at, expires_at
		FROM polls
		WHERE deleted_at IS NULL
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all polls: %w", err)
	}
	defer rows.Close()

	return r.scanPolls(ctx, rows)
}

func (r *pollRepository) List(ctx context.Context, limit, offset int) ([]*domain.Poll, error) {
	query := `
		SELECT p.id, p.title, p.description, p.created_at, p.expires_at
		FROM polls p
		LEFT JOIN poll_results pr ON p.id = pr.poll_id
		WHERE p.deleted_at IS NULL
		GROUP BY p.id
		ORDER BY COALESCE(SUM(pr.vote_count), 0) DESC, p.created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list polls: %w", err)
	}
	defer rows.Close()

	return r.scanPolls(ctx, rows)
}

func (r *pollRepository) Search(ctx context.Context, limit, offset int, q string) ([]*domain.Poll, error) {
	query := `
		SELECT p.id, p.title, p.description, p.created_at, p.expires_at
		FROM polls p
		LEFT JOIN poll_results pr ON p.id = pr.poll_id
		WHERE p.deleted_at IS NULL AND p.title ILIKE $1
		GROUP BY p.id
		ORDER BY COALESCE(SUM(pr.vote_count), 0) DESC, p.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, "%"+q+"%", limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search polls: %w", err)
	}
	defer rows.Close()

	return r.scanPolls(ctx, rows)
}

func (r *pollRepository) scanPolls(ctx context.Context, rows *sql.Rows) ([]*domain.Poll, error) {
	var polls []*domain.Poll
	for rows.Next() {
		var poll domain.Poll
		if err := rows.Scan(&poll.ID, &poll.Title, &poll.Description, &poll.CreatedAt, &poll.ExpiresAt); err != nil {
			return nil, fmt.Errorf("failed to scan poll: %w", err)
		}

		options, err := r.fetchOptions(ctx, poll.ID)
		if err != nil {
			return nil, err
		}
		poll.Options = options

		polls = append(polls, &poll)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating polls: %w", err)
	}
	return polls, nil
}

func (r *pollRepository) fetchOptions(ctx context.Context, pollID uuid.UUID) ([]domain.PollOption, error) {
	queryOptions := `
		SELECT id, poll_id, text, created_at
		FROM poll_options
		WHERE poll_id = $1
	`
	rows, err := r.db.QueryContext(ctx, queryOptions, pollID)
	if err != nil {
		return nil, fmt.Errorf("failed to get poll options: %w", err)
	}
	defer rows.Close()

	var options []domain.PollOption
	for rows.Next() {
		var opt domain.PollOption
		if err := rows.Scan(&opt.ID, &opt.PollID, &opt.Text, &opt.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan option: %w", err)
		}
		options = append(options, opt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating options: %w", err)
	}
	return options, nil
}

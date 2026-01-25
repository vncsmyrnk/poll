package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/vncsmyrnk/poll/internal/core/domain"
	"github.com/vncsmyrnk/poll/internal/core/ports"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) ports.UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, name, created_at FROM users WHERE email = $1 AND deleted_at IS NULL`
	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, email, name, created_at FROM users WHERE id = $1 AND deleted_at IS NULL`
	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users (email, name) VALUES ($1, $2) RETURNING id, created_at`
	err := r.db.QueryRowContext(ctx, query, user.Email, user.Name).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return err
	}
	return nil
}

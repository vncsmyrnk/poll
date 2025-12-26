package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/poll/api/internal/core/domain"
	"github.com/poll/api/internal/core/ports"
)

type AuthRepository struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) ports.AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) StoreRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, revoked)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query, token.UserID, token.TokenHash, token.ExpiresAt, token.Revoked).Scan(&token.ID, &token.CreatedAt)
}

func (r *AuthRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`
	token := &domain.RefreshToken{}
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.Revoked,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return token, nil
}

func (r *AuthRepository) RevokeRefreshToken(ctx context.Context, id string) error {
	query := `UPDATE refresh_tokens SET revoked = true WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

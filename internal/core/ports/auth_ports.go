package ports

import (
	"context"

	"github.com/poll/api/internal/core/domain"
)

type AuthRepository interface {
	StoreRefreshToken(ctx context.Context, token *domain.RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id string) error
}

type AuthService interface {
	LoginWithGoogle(ctx context.Context, googleToken string) (string, string, error) // returns access_token, refresh_token, error
	RefreshAccessToken(ctx context.Context, refreshToken string) (string, string, error) // returns new access_token, new refresh_token (optional, implementation specific)
}

package ports

import (
	"context"

	"github.com/poll/api/internal/core/domain"
)

type TokenVerifier interface {
	Verify(ctx context.Context, token string, clientID string) (*TokenPayload, error)
}

type TokenPayload struct {
	Email string
	Name  string
}

type AuthRepository interface {
	StoreRefreshToken(ctx context.Context, token *domain.RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id string) error
}

type AuthService interface {
	LoginWithGoogle(ctx context.Context, googleToken string) (string, string, error)
	RefreshAccessToken(ctx context.Context, refreshToken string) (string, string, error)
	Logout(ctx context.Context, refreshToken string) error
}

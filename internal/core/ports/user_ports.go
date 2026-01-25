package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/vncsmyrnk/poll/internal/core/domain"
)

type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
}

type UserService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/poll/api/internal/core/domain"
)

type UserService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

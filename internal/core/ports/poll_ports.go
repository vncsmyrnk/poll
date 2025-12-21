package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/poll/api/internal/core/domain"
)

type PollRepository interface {
	Save(ctx context.Context, poll *domain.Poll) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Poll, error)
	GetAll(ctx context.Context) ([]*domain.Poll, error)
	List(ctx context.Context, limit, offset int) ([]*domain.Poll, error)
	Search(ctx context.Context, limit, offset int, query string) ([]*domain.Poll, error)
}

type CreatePollInput struct {
	Title       string
	Description string
	Options     []string
}

type ListPollsInput struct {
	Page  int
	Query string
}

type PollService interface {
	Create(ctx context.Context, input CreatePollInput) (*domain.Poll, error)
	GetPoll(ctx context.Context, id string) (*domain.Poll, error)
	ListPolls(ctx context.Context, input ListPollsInput) ([]*domain.Poll, error)
}

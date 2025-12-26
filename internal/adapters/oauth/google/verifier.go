package google

import (
	"context"
	"errors"

	"github.com/poll/api/internal/core/ports"
	"google.golang.org/api/idtoken"
)

type GoogleVerifier struct{}

func NewVerifier() ports.TokenVerifier {
	return &GoogleVerifier{}
}

func (v *GoogleVerifier) Verify(ctx context.Context, token string, clientID string) (*ports.TokenPayload, error) {
	payload, err := idtoken.Validate(ctx, token, clientID)
	if err != nil {
		return nil, err
	}
	email, ok := payload.Claims["email"].(string)
	if !ok {
		return nil, errors.New("email not found in claims")
	}
	name, ok := payload.Claims["name"].(string)
	if !ok {
		return nil, errors.New("name not found in claims")
	}
	return &ports.TokenPayload{Email: email, Name: name}, nil
}

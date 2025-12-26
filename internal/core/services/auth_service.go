package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/poll/api/internal/core/domain"
	"github.com/poll/api/internal/core/ports"
)

type AuthService struct {
	userRepo            ports.UserRepository
	authRepo            ports.AuthRepository
	googleTokenVerifier ports.TokenVerifier
	jwtSecret           []byte
	googleClientID      string
}

func NewAuthService(userRepo ports.UserRepository, authRepo ports.AuthRepository, googleTokenVerifier ports.TokenVerifier) *AuthService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		fmt.Println("Warning: JWT_SECRET not set")
	}

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	return &AuthService{
		userRepo:            userRepo,
		authRepo:            authRepo,
		googleTokenVerifier: googleTokenVerifier,
		jwtSecret:           []byte(secret),
		googleClientID:      clientID,
	}
}

func (s *AuthService) LoginWithGoogle(ctx context.Context, googleToken string) (string, string, error) {
	payload, err := s.googleTokenVerifier.Verify(ctx, googleToken, s.googleClientID)
	if err != nil {
		return "", "", fmt.Errorf("invalid google token: %w", err)
	}

	return s.login(ctx, payload.Email, payload.Name)
}

func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (string, string, error) {
	tokenHash := s.hashToken(refreshToken)

	rtEntity, err := s.authRepo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return "", "", fmt.Errorf("failed to get refresh token: %w", err)
	}
	if rtEntity == nil {
		return "", "", errors.New("refresh token not found")
	}

	if rtEntity.Revoked {
		return "", "", errors.New("refresh token revoked")
	}
	if rtEntity.ExpiresAt.Before(time.Now()) {
		return "", "", errors.New("refresh token expired")
	}

	user, err := s.userRepo.GetByID(ctx, rtEntity.UserID.String())
	if err != nil {
		return "", "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return "", "", errors.New("user not found")
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Optional: Rotate Refresh Token
	// For simplicity, keep the same refresh token until expiry, or we can rotate it.
	// Let's keep it simple for now and return the same refresh token, or empty if we don't rotate.

	return accessToken, refreshToken, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := s.hashToken(refreshToken)

	rtEntity, err := s.authRepo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to get refresh token: %w", err)
	}
	if rtEntity == nil {
		return nil
	}

	return s.authRepo.RevokeRefreshToken(ctx, rtEntity.ID.String())
}

func (s *AuthService) login(ctx context.Context, email, name string) (string, string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		user = &domain.User{
			Email: email,
			Name:  name,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return "", "", fmt.Errorf("failed to create user: %w", err)
		}
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	refreshTokenHash := s.hashToken(refreshToken)

	rtEntity := &domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
		Revoked:   false,
	}

	if err := s.authRepo.StoreRefreshToken(ctx, rtEntity); err != nil {
		return "", "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) generateAccessToken(user *domain.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":   user.ID.String(),
		"email": user.Email,
		"exp":   time.Now().Add(15 * time.Minute).Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (s *AuthService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

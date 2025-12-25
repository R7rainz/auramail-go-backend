package auth

import (
	"context"
	"errors"

	"github.com/r7rainz/auramail/internal/user"
)

var ErrInvalidRefreshToken = errors.New("invalid refresh token")

type Service struct {
	users user.Repository
}

func NewService(users user.Repository) *Service {
	return &Service{users: users}
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (string, error) {
	_, err := ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", ErrInvalidRefreshToken
	}

	u, err := s.users.FindByRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", ErrInvalidRefreshToken
	}
	
	accessToken, err := GenerateAccessToken(u.ID, u.Email, u.Name)
	if err != nil {
		return "", err
	}
	return accessToken, nil
}

func (s *Service) Logout(ctx context.Context, userID int) error {
	return s.users.ClearRefreshToken(ctx, userID)
}


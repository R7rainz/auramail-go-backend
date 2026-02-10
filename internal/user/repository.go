package user

import (
	"context"
)

type Repository interface {
	FindOrCreateGoogleUser(
		ctx context.Context,
		email string,
		name string,
		googleSub string,
	) (*User, error)

	UpdateRefreshToken(
		ctx context.Context,
		userID int,
		refreshToken string,
	) error

	FindByRefreshToken(ctx context.Context, token string) (*User, error)

	ClearRefreshToken(ctx context.Context, userID int) error

	FindByID(ctx context.Context, id int) (*User, error)

	Save(ctx context.Context, user *User) error
}

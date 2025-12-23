package user

import "context"

type Repository interface {
	FindOrCreateGoogleUser(
		ctx context.Context,
		email string,
		name string,
		googleSub string,
	) (*User, error)
}

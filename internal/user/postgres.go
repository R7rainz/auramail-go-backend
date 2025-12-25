package user

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) FindOrCreateGoogleUser(
	ctx context.Context,
	email string,
	name string,
	googleSub string,
) (*User, error) {

	//TODO: write sql + scan here
	query := `INSERT INTO users (email, name, provider, provider_id) VALUES ($1, $2, 'google', $3) ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name RETURNING id, email, name, provider, provider_id;`

	var u User
	err := r.db.QueryRow(ctx, query, email, name, googleSub).Scan(&u.ID, &u.Email, &u.Name, &u.Provider, &u.ProviderID)

	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (r *PostgresRepository) UpdateRefreshToken(
	ctx context.Context, 
	userID int, 
	refreshToken string,
) error {
	query := `UPDATE users SET refresh_token = $1 WHERE id = $2;`

	_, err := r.db.Exec(ctx, query, refreshToken, userID)
	if err != nil {
		return fmt.Errorf("failed to update refresh token for user %d: %w", userID, err)
	}

	return nil
}


func (r *PostgresRepository) FindByRefreshToken(ctx context.Context, token string) (*User, error) {
	query := `SELECT id, email, name, provider, provider_id, refresh_token FROM users WHERE refresh_token = $1;`

	var u User
	err := r.db.QueryRow(ctx, query, token).Scan(&u.ID, &u.Email, &u.Name, &u.Provider, &u.ProviderID, &u.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by refresh token: %w", err)
	}

	return &u, nil
}

func (r *PostgresRepository) ClearRefreshToken(ctx context.Context, userID int) error {
	query := `UPDATE users SET refresh_token = NULL WHERE id = $1;`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to clear refresh token for user %d: %w", userID, err)
	}
	
	return nil
}

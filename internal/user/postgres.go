package user

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

// FindByID implements [Repository].
// func (r *PostgresRepository) FindByID(ctx context.Context, id string) (*User, error) {
// 	log.Printf("DEBUG: Searching for User ID %s in database...", id)
// 	query := `SELECT id, email, name, refresh_token, FROM users WHERE id = $1;`
//
// 	var u User
// 	err := r.db.QueryRow(ctx, query, id).Scan(
// 		&u.ID,
// 		&u.Email,
// 		&u.Name,
// 		&u.Provider, 
// 		&u.ProviderID,
// 		&u.RefreshToken,
// 	)
//
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to find user by ID %s: %w", id, err)
// 	}
// 	return &u, nil
// }

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (*User, error) {
    // 1. Log exactly what we are looking for
    log.Printf("DB QUERY: Looking for ID [%s] as an integer", id)

    var u User
    // 2. Use the ::int cast to ensure Postgres compares correctly
    query := `SELECT id, email, name, provider, provider_id, refresh_token 
              FROM users WHERE id = $1::int;`

    err := r.db.QueryRow(ctx, query, id).Scan(
        &u.ID, &u.Email, &u.Name, &u.Provider, &u.ProviderID, &u.RefreshToken,
    )

    if err != nil {
        // 3. Log the ACTUAL database error
        log.Printf("DB ERROR for ID %s: %v", id, err)
        return nil, err
    }

    log.Printf("DB SUCCESS: Found user %s", u.Email)
    return &u, nil
}

// Save implements [Repository].
func (r *PostgresRepository) Save(ctx context.Context, user *User) error {
	query := `UPDATE users SET email = $1, name = $2, refresh_token = $3 WHERE id = $4;`
	
	_, err := r.db.Exec(ctx, query, user.Email, user.Name, user.RefreshToken, user.ID)
	return err
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) FindOrCreateGoogleUser(ctx context.Context, email, name, sub string) (*User, error) {
    var u User
    query := `SELECT id, email, name FROM users WHERE email = $1`
    err := r.db.QueryRow(ctx, query, email).Scan(&u.ID, &u.Email, &u.Name)

    if err == nil {
        return &u, nil
    }

    insertQuery := `INSERT INTO users (email, name, provider_id) 
                    VALUES ($1, $2, $3) 
                    RETURNING id, email, name`
    err = r.db.QueryRow(ctx, insertQuery, email, name, sub).Scan(&u.ID, &u.Email, &u.Name)
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

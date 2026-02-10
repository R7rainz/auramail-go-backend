package user

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/r7rainz/auramail/internal/ai"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func (r *PostgresRepository) FindByID(ctx context.Context, id int) (*User, error) {
	log.Printf("DB QUERY: Looking for ID [%d]", id)

	var u User
	query := `SELECT id, email, name, provider, provider_id, refresh_token 
			  FROM users WHERE id = $1;`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.Name, &u.Provider, &u.ProviderID, &u.RefreshToken,
	)

	if err != nil {
		log.Printf("DB ERROR for ID %d: %v", id, err)
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

func (r *PostgresRepository) GetSummary(ctx context.Context, gmailID string) (*ai.AIResult, error) {
	var data []byte
	query := `SELECT data FROM email_summaries WHERE gmail_id = $1`

	err := r.db.QueryRow(ctx, query, gmailID).Scan(&data)
	if err != nil {
		return nil, err
	}

	var result ai.AIResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached summary: %w", err)
	}

	return &result, nil
}

func (r *PostgresRepository) SaveSummary(ctx context.Context, userID int, gmailID string, res *ai.AIResult) error {
	jsonData, err := json.Marshal(res)
	if err != nil {
		return fmt.Errorf("failed to unmarshal AI result: %w", err)
	}

	query := `
		INSERT INTO email_summaries (user_id, gmail_id, category, company, role, summary, deadline, apply_link, data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (gmail_id) DO NOTHING`

	_, err = r.db.Exec(ctx, query,
		userID,
		gmailID,
		res.Category,
		res.Company,
		res.Role,
		res.Summary,
		res.Deadline,
		res.ApplyLink,
		jsonData,
	)
	if err != nil {
		return fmt.Errorf("failed to save summary to db: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetSummariesByQuery(ctx context.Context, userID int, searchQuery string) ([]*ai.AIResult, error) {
	query := `SELECT data FROM email_summaries WHERE user_id = $1 AND (summary ILIKE $2 OR company ILIKE $2 OR role ILIKE $2) ORDER BY created_at DESC LIMIT 50`
	rows, err := r.db.Query(ctx, query, userID, "%"+searchQuery+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*ai.AIResult
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var res ai.AIResult
		json.Unmarshal(data, &res)
		results = append(results, &res)
	}
	return results, nil
}

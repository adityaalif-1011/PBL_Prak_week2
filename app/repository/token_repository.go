package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/model"
)

var ErrTokenNotFound = errors.New("token not found")

type TokenRepository struct {
	pool *pgxpool.Pool
}

func NewTokenRepository(pool *pgxpool.Pool) *TokenRepository {
	return &TokenRepository{pool: pool}
}

// Save - simpan refresh token (hash-nya)
func (r *TokenRepository) Save(ctx context.Context, t model.RefreshToken) error {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
		t.UserID, t.TokenHash, t.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	return nil
}

// FindActive - cari refresh token yang masih aktif
func (r *TokenRepository) FindActive(ctx context.Context, tokenHash string) (model.RefreshToken, error) {
	var t model.RefreshToken
	var revokedAt *string

	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, revoked_at, created_at 
		 FROM refresh_tokens 
		 WHERE token_hash = $1 
		   AND revoked_at IS NULL 
		   AND expires_at > NOW()`,
		tokenHash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &revokedAt, &t.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.RefreshToken{}, ErrTokenNotFound
		}
		return model.RefreshToken{}, fmt.Errorf("find token: %w", err)
	}
	return t, nil
}

// Revoke - cabut refresh token
func (r *TokenRepository) Revoke(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL",
		tokenHash,
	)
	if err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}
	return nil
}

// RevokeAllForUser - cabut semua token milik user
func (r *TokenRepository) RevokeAllForUser(ctx context.Context, userID int) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL",
		userID,
	)
	if err != nil {
		return fmt.Errorf("revoke all: %w", err)
	}
	return nil
}

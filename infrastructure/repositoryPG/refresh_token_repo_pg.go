package repositoryPG

import (
	"context"
	"database/sql"
	"expense_tracker/domain"
)

type RefreshTokenRepoPG struct {
	DB *sql.DB
}

func NewRefreshTokenRepoPG(db *sql.DB) *RefreshTokenRepoPG {
	return &RefreshTokenRepoPG{DB: db}
}

func (r *RefreshTokenRepoPG) Create(ctx context.Context, token *domain.RefreshToken) error {
	query := `INSERT INTO refresh_tokens (token_id, user_id, token_hash, expires_at, revoked_at, created_at)
	VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.DB.ExecContext(ctx, query, token.TokenID, token.UserID, token.TokenHash, token.ExpiresAt, token.RevokedAt, token.CreatedAt)
	return err
}

func (r *RefreshTokenRepoPG) GetActiveByTokenIDAndHash(ctx context.Context, tokenID string, tokenHash string) (*domain.RefreshToken, error) {
	query := `SELECT token_id, user_id, token_hash, expires_at, revoked_at, created_at
	FROM refresh_tokens
	WHERE token_id = $1 AND token_hash = $2 AND revoked_at IS NULL AND expires_at > NOW()`

	var token domain.RefreshToken
	var revokedAt sql.NullTime
	err := r.DB.QueryRowContext(ctx, query, tokenID, tokenHash).
		Scan(&token.TokenID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &revokedAt, &token.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if revokedAt.Valid {
		token.RevokedAt = &revokedAt.Time
	}
	return &token, nil
}

func (r *RefreshTokenRepoPG) RevokeByTokenID(ctx context.Context, tokenID string) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_id = $1 AND revoked_at IS NULL`
	_, err := r.DB.ExecContext(ctx, query, tokenID)
	return err
}

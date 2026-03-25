package repository

import (
	"context"
	"expense_tracker/domain"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	GetActiveByTokenIDAndHash(ctx context.Context, tokenID string, tokenHash string) (*domain.RefreshToken, error)
	RevokeByTokenID(ctx context.Context, tokenID string) error
}

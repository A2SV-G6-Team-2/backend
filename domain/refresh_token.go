package domain

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	TokenID   string     `json:"token_id"`
	UserID    uuid.UUID  `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

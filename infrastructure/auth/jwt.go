package auth

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
)

type JWTService struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func NewJWTService(secret string) *JWTService {
	return &JWTService{
		Secret:     secret,
		AccessTTL:  readEnvDurationHours("ACCESS_TOKEN_TTL_HOURS", 10),
		RefreshTTL: readEnvDurationHours("REFRESH_TOKEN_TTL_HOURS", 168),
	}
}

func (j JWTService) Generate(userID uuid.UUID) (string, error) {
	return j.GenerateAccessToken(userID)
}

func (j JWTService) GenerateAccessToken(userID uuid.UUID) (string, error) {
	return j.generateToken(userID, "access", j.AccessTTL)
}

func (j JWTService) GenerateTokenPair(userID uuid.UUID) (string, string, string, error) {
	tokenID := uuid.New().String()

	accessToken, err := j.GenerateAccessToken(userID)
	if err != nil {
		return "", "", "", err
	}

	refreshToken, err := j.generateRefreshToken(userID, tokenID)
	if err != nil {
		return "", "", "", err
	}

	return accessToken, refreshToken, tokenID, nil
}

func (j JWTService) Validate(tokenStr string) (uuid.UUID, error) {
	return j.validateToken(tokenStr, "access")
}

func (j JWTService) ValidateRefreshToken(tokenStr string) (uuid.UUID, error) {
	userID, _, err := j.ParseRefreshToken(tokenStr)
	if err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func (j JWTService) ParseRefreshToken(tokenStr string) (uuid.UUID, string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(j.Secret), nil
	})
	if err != nil || !token.Valid {
		if err == nil {
			err = errors.New("invalid token")
		}
		return uuid.Nil, "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, "", errors.New("invalid token claims")
	}

	tokenType, ok := claims["token_type"].(string)
	if !ok || tokenType != "refresh" {
		return uuid.Nil, "", errors.New("invalid token type")
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return uuid.Nil, "", errors.New("invalid token claims")
	}

	tokenID, ok := claims["jti"].(string)
	if !ok || tokenID == "" {
		return uuid.Nil, "", errors.New("invalid token claims")
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, "", err
	}

	return parsedUserID, tokenID, nil
}

func (j JWTService) generateToken(userID uuid.UUID, tokenType string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id":    userID.String(),
		"token_type": tokenType,
		"exp":        time.Now().Add(ttl).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.Secret))
}

func (j JWTService) generateRefreshToken(userID uuid.UUID, tokenID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":    userID.String(),
		"token_type": "refresh",
		"jti":        tokenID,
		"exp":        time.Now().Add(j.RefreshTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.Secret))
}

func (j JWTService) validateToken(tokenStr string, expectedType string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(j.Secret), nil
	})
	if err != nil || !token.Valid {
		if err == nil {
			err = errors.New("invalid token")
		}
		return uuid.Nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, errors.New("invalid token claims")
	}

	tokenType, ok := claims["token_type"].(string)
	if !ok || tokenType != expectedType {
		return uuid.Nil, errors.New("invalid token type")
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return uuid.Nil, errors.New("invalid token claims")
	}

	return uuid.Parse(userID)
}

func readEnvDurationHours(key string, fallback int) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return time.Duration(fallback) * time.Hour
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return time.Duration(fallback) * time.Hour
	}

	return time.Duration(value) * time.Hour
}

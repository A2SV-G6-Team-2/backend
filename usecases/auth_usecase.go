package usecases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"expense_tracker/domain"
	"expense_tracker/repository"
	"regexp"
	"time"

	"github.com/google/uuid"
)

var (
	uppercasePattern = regexp.MustCompile(`[A-Z]`)
	lowercasePattern = regexp.MustCompile(`[a-z]`)
	digitPattern     = regexp.MustCompile(`[0-9]`)
	specialPattern   = regexp.MustCompile(`[^A-Za-z0-9]`)
)

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(password string, hash string) error
}

type JWTService interface {
	Generate(uuid.UUID) (string, error)
	GenerateTokenPair(uuid.UUID) (string, string, string, error)
	ValidateRefreshToken(string) (uuid.UUID, error)
	ParseRefreshToken(string) (uuid.UUID, string, error)
}

type AuthUsecase interface {
	Register(ctx context.Context, input RegisterInput) (domain.User, error)
	Login(ctx context.Context, input LoginInput) (AuthResponse, error)
	Refresh(ctx context.Context, input RefreshInput) (AuthResponse, error)
	Logout(ctx context.Context, input LogoutInput) error
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type RefreshInput struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutInput struct {
	RefreshToken string `json:"refresh_token"`
}

type AuthResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	User         domain.User `json:"user"`
}

type authUsecase struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	hasher           PasswordHasher
	jwt              JWTService
}

func NewAuthUsecase(r repository.UserRepository, refreshTokenRepo repository.RefreshTokenRepository, h PasswordHasher, j JWTService) AuthUsecase {
	return &authUsecase{
		userRepo:         r,
		refreshTokenRepo: refreshTokenRepo,
		hasher:           h,
		jwt:              j,
	}
}

// Register and Login implementation
func (a *authUsecase) Register(ctx context.Context, in RegisterInput) (domain.User, error) {
	exists, _ := a.userRepo.GetByEmail(ctx, in.Email)
	if exists != nil {
		return domain.User{}, errors.New("email already used")
	}

	if err := validatePassword(in.Password); err != nil {
		return domain.User{}, err
	}

	hash, err := a.hasher.Hash(in.Password)
	if err != nil {
		return domain.User{}, err
	}

	user := domain.User{
		UserID:          uuid.New(),
		Name:            in.Name,
		Email:           in.Email,
		PasswordHash:    hash,
		BudgetingStyle:  "flexible",
		DefaultCurrency: "ETB",
	}

	return user, a.userRepo.Create(ctx, &user)
}

func (a *authUsecase) Login(ctx context.Context, in LoginInput) (AuthResponse, error) {
	user, err := a.userRepo.GetByEmail(ctx, in.Email)
	if err != nil || user == nil {
		return AuthResponse{}, errors.New("invalid credentials")
	}

	if err := a.hasher.Compare(in.Password, user.PasswordHash); err != nil {
		return AuthResponse{}, errors.New("invalid credentials")
	}

	accessToken, refreshToken, tokenID, err := a.jwt.GenerateTokenPair(user.UserID)
	if err != nil {
		return AuthResponse{}, err
	}

	now := time.Now().UTC()
	if err := a.refreshTokenRepo.Create(ctx, &domain.RefreshToken{
		TokenID:   tokenID,
		UserID:    user.UserID,
		TokenHash: hashToken(refreshToken),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
	}); err != nil {
		return AuthResponse{}, err
	}

	user.PasswordHash = ""
	return AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	}, nil
}

func validatePassword(password string) error {
	switch {
	case len(password) < 8:
		return errors.New("password must be at least 8 characters")
	case !uppercasePattern.MatchString(password):
		return errors.New("password must contain at least one uppercase letter")
	case !lowercasePattern.MatchString(password):
		return errors.New("password must contain at least one lowercase letter")
	case !digitPattern.MatchString(password):
		return errors.New("password must contain at least one number")
	case !specialPattern.MatchString(password):
		return errors.New("password must contain at least one special character")
	default:
		return nil
	}
}

func (a *authUsecase) Refresh(ctx context.Context, in RefreshInput) (AuthResponse, error) {
	if in.RefreshToken == "" {
		return AuthResponse{}, errors.New("refresh token is required")
	}

	userID, currentTokenID, err := a.jwt.ParseRefreshToken(in.RefreshToken)
	if err != nil {
		return AuthResponse{}, errors.New("invalid refresh token")
	}

	storedToken, err := a.refreshTokenRepo.GetActiveByTokenIDAndHash(ctx, currentTokenID, hashToken(in.RefreshToken))
	if err != nil || storedToken == nil {
		return AuthResponse{}, errors.New("invalid refresh token")
	}

	user, err := a.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return AuthResponse{}, errors.New("user not found")
	}

	accessToken, refreshToken, newTokenID, err := a.jwt.GenerateTokenPair(user.UserID)
	if err != nil {
		return AuthResponse{}, err
	}

	now := time.Now().UTC()
	if err := a.refreshTokenRepo.Create(ctx, &domain.RefreshToken{
		TokenID:   newTokenID,
		UserID:    user.UserID,
		TokenHash: hashToken(refreshToken),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
	}); err != nil {
		return AuthResponse{}, err
	}

	if err := a.refreshTokenRepo.RevokeByTokenID(ctx, currentTokenID); err != nil {
		return AuthResponse{}, err
	}

	user.PasswordHash = ""
	return AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	}, nil
}

func (a *authUsecase) Logout(ctx context.Context, in LogoutInput) error {
	if in.RefreshToken == "" {
		return errors.New("refresh token is required")
	}

	_, tokenID, err := a.jwt.ParseRefreshToken(in.RefreshToken)
	if err != nil {
		return errors.New("invalid refresh token")
	}

	storedToken, err := a.refreshTokenRepo.GetActiveByTokenIDAndHash(ctx, tokenID, hashToken(in.RefreshToken))
	if err != nil || storedToken == nil {
		return errors.New("invalid refresh token")
	}

	return a.refreshTokenRepo.RevokeByTokenID(ctx, tokenID)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"expense_tracker/domain"
	deliveryhttp "expense_tracker/delivery/http"
	"expense_tracker/infrastructure/auth"
	"expense_tracker/usecases"

	"github.com/google/uuid"
)

type fakePasswordHasher struct{}

func (fakePasswordHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (fakePasswordHasher) Compare(password string, hash string) error {
	if hash != "hashed:"+password {
		return errTest("password mismatch")
	}
	return nil
}

type errTest string

func (e errTest) Error() string { return string(e) }

type fakeUserRepo struct {
	byEmail map[string]*domain.User
	byID    map[uuid.UUID]*domain.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byEmail: map[string]*domain.User{},
		byID:    map[uuid.UUID]*domain.User{},
	}
}

func (r *fakeUserRepo) Create(_ context.Context, user *domain.User) error {
	copy := *user
	r.byEmail[user.Email] = &copy
	r.byID[user.UserID] = &copy
	return nil
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	user := r.byEmail[email]
	if user == nil {
		return nil, nil
	}
	copy := *user
	return &copy, nil
}

func (r *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	user := r.byID[id]
	if user == nil {
		return nil, nil
	}
	copy := *user
	return &copy, nil
}

func (r *fakeUserRepo) Update(_ context.Context, user *domain.User) error {
	copy := *user
	r.byEmail[user.Email] = &copy
	r.byID[user.UserID] = &copy
	return nil
}

type fakeRefreshTokenRepo struct {
	records map[string]*domain.RefreshToken
}

func newFakeRefreshTokenRepo() *fakeRefreshTokenRepo {
	return &fakeRefreshTokenRepo{records: map[string]*domain.RefreshToken{}}
}

func (r *fakeRefreshTokenRepo) Create(_ context.Context, token *domain.RefreshToken) error {
	copy := *token
	r.records[token.TokenID] = &copy
	return nil
}

func (r *fakeRefreshTokenRepo) GetActiveByTokenIDAndHash(_ context.Context, tokenID string, tokenHash string) (*domain.RefreshToken, error) {
	record := r.records[tokenID]
	if record == nil || record.TokenHash != tokenHash || record.RevokedAt != nil {
		return nil, nil
	}
	copy := *record
	return &copy, nil
}

func (r *fakeRefreshTokenRepo) RevokeByTokenID(_ context.Context, tokenID string) error {
	record := r.records[tokenID]
	if record == nil {
		return nil
	}
	now := record.CreatedAt
	record.RevokedAt = &now
	return nil
}

type fakeAuthUsecase struct {
	registerFn func(context.Context, usecases.RegisterInput) (domain.User, error)
	loginFn    func(context.Context, usecases.LoginInput) (usecases.AuthResponse, error)
	refreshFn  func(context.Context, usecases.RefreshInput) (usecases.AuthResponse, error)
	logoutFn   func(context.Context, usecases.LogoutInput) error
}

func (f fakeAuthUsecase) Register(ctx context.Context, in usecases.RegisterInput) (domain.User, error) {
	return f.registerFn(ctx, in)
}
func (f fakeAuthUsecase) Login(ctx context.Context, in usecases.LoginInput) (usecases.AuthResponse, error) {
	return f.loginFn(ctx, in)
}
func (f fakeAuthUsecase) Refresh(ctx context.Context, in usecases.RefreshInput) (usecases.AuthResponse, error) {
	return f.refreshFn(ctx, in)
}
func (f fakeAuthUsecase) Logout(ctx context.Context, in usecases.LogoutInput) error {
	return f.logoutFn(ctx, in)
}

func TestAuthUsecaseRegisterRejectsWeakPassword(t *testing.T) {
	userRepo := newFakeUserRepo()
	refreshRepo := newFakeRefreshTokenRepo()
	jwtSvc := auth.NewJWTService("test-secret")
	uc := usecases.NewAuthUsecase(userRepo, refreshRepo, fakePasswordHasher{}, jwtSvc)

	_, err := uc.Register(contextBackground(), usecases.RegisterInput{
		Name:     "Mike",
		Email:    "mike@example.com",
		Password: "weak",
	})
	if err == nil {
		t.Fatal("expected password validation error")
	}
	if !strings.Contains(err.Error(), "password must be at least 8 characters") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthUsecaseLoginRefreshAndLogoutFlow(t *testing.T) {
	userRepo := newFakeUserRepo()
	refreshRepo := newFakeRefreshTokenRepo()
	jwtSvc := auth.NewJWTService("test-secret")
	uc := usecases.NewAuthUsecase(userRepo, refreshRepo, fakePasswordHasher{}, jwtSvc)

	userID := uuid.New()
	if err := userRepo.Create(contextBackground(), &domain.User{
		UserID:          userID,
		Name:            "Mike",
		Email:           "mike@example.com",
		PasswordHash:    "hashed:Secure123!",
		BudgetingStyle:  "flexible",
		DefaultCurrency: "ETB",
	}); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	loginResp, err := uc.Login(contextBackground(), usecases.LoginInput{
		Email:    "mike@example.com",
		Password: "Secure123!",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	_, firstTokenID, err := jwtSvc.ParseRefreshToken(loginResp.RefreshToken)
	if err != nil {
		t.Fatalf("parse refresh token: %v", err)
	}

	refreshResp, err := uc.Refresh(contextBackground(), usecases.RefreshInput{RefreshToken: loginResp.RefreshToken})
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshResp.RefreshToken == "" || refreshResp.RefreshToken == loginResp.RefreshToken {
		t.Fatal("expected rotated refresh token")
	}
	if refreshRepo.records[firstTokenID] == nil || refreshRepo.records[firstTokenID].RevokedAt == nil {
		t.Fatal("expected old refresh token to be revoked")
	}

	if err := uc.Logout(contextBackground(), usecases.LogoutInput{RefreshToken: refreshResp.RefreshToken}); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := uc.Refresh(contextBackground(), usecases.RefreshInput{RefreshToken: refreshResp.RefreshToken}); err == nil {
		t.Fatal("expected revoked refresh token to fail")
	}
}

func TestAuthHandlerLoginAndLogoutEnvelopes(t *testing.T) {
	handler := deliveryhttp.NewAuthHandler(fakeAuthUsecase{
		registerFn: func(context.Context, usecases.RegisterInput) (domain.User, error) { return domain.User{}, nil },
		loginFn: func(_ context.Context, _ usecases.LoginInput) (usecases.AuthResponse, error) {
			return usecases.AuthResponse{AccessToken: "access", RefreshToken: "refresh"}, nil
		},
		refreshFn: func(context.Context, usecases.RefreshInput) (usecases.AuthResponse, error) {
			return usecases.AuthResponse{}, nil
		},
		logoutFn: func(context.Context, usecases.LogoutInput) error { return nil },
	})

	loginRec := httptest.NewRecorder()
	loginReq := newJSONRequest(t, http.MethodPost, "/auth/login", map[string]string{
		"email":    "mike@example.com",
		"password": "Secure123!",
	})
	handler.Login(loginRec, loginReq)
	loginEnv := decodeEnvelope(t, loginRec)
	if loginRec.Code != http.StatusOK || !loginEnv.Success || loginEnv.Message != "User logged in successfully" {
		t.Fatalf("unexpected login response: code=%d env=%+v", loginRec.Code, loginEnv)
	}

	logoutRec := httptest.NewRecorder()
	logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader("{"))
	logoutReq.Header.Set("Content-Type", "application/json")
	handler.Logout(logoutRec, logoutReq)
	logoutEnv := decodeEnvelope(t, logoutRec)
	if logoutRec.Code != http.StatusBadRequest || logoutEnv.Success || len(logoutEnv.Errors) != 1 {
		t.Fatalf("unexpected logout response: code=%d env=%+v", logoutRec.Code, logoutEnv)
	}
}

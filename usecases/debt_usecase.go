package usecases

import (
	"context"
	"errors"
	"expense_tracker/domain"
	"expense_tracker/repository"
	"time"

	"github.com/google/uuid"
)

var (
	ErrDebtIDRequired       = errors.New("debt id is required")
	ErrUserIDRequired       = errors.New("user id is required")
	ErrDebtTypeRequired     = errors.New("debt type is required")
	ErrPeerNameRequired     = errors.New("peer name is required")
	ErrAmountMustBePositive = errors.New("amount must be positive")
	ErrDueDateInPast        = errors.New("due date cannot be in the past")
	ErrDebtAlreadyPaid      = errors.New("debt is already paid")
)

type DebtUsecase struct {
	repo repository.DebtRepository
	now  func() time.Time
}

func NewDebtUsecase(repo repository.DebtRepository) *DebtUsecase {
	return &DebtUsecase{
		repo: repo,
		now:  time.Now,
	}
}

func (u *DebtUsecase) Create(ctx context.Context, debt *domain.Debt) error {
	if debt == nil {
		return errors.New("debt is required")
	}
	if debt.ID == "" {
		// Generate an ID when the client doesn't provide one. This keeps
		// create calls ergonomic from clients while preserving the
		// requirement that other usecase methods operate on an ID.
		debt.ID = uuid.New().String()
	}
	if debt.UserID == "" {
		return ErrUserIDRequired
	}
	if debt.Type == "" {
		return ErrDebtTypeRequired
	}
	if debt.PeerName == "" {
		return ErrPeerNameRequired
	}
	if debt.Amount <= 0 {
		return ErrAmountMustBePositive
	}
	if isDateInPast(debt.DueDate, u.now().UTC()) {
		return ErrDueDateInPast
	}
	if debt.Status == "" {
		debt.Status = domain.DebtStatusPending
	}
	debt.CreatedAt = u.now().UTC()

	return u.repo.Create(ctx, debt)
}

func (u *DebtUsecase) Update(ctx context.Context, debt *domain.Debt) error {
	if debt == nil {
		return errors.New("debt is required")
	}
	if debt.ID == "" {
		return ErrDebtIDRequired
	}

	existing, err := u.repo.GetByID(ctx, debt.ID)
	if err != nil {
		return err
	}
	if existing.Status == domain.DebtStatusPaid {
		return ErrDebtAlreadyPaid
	}

	if debt.Type == "" {
		return ErrDebtTypeRequired
	}
	if debt.PeerName == "" {
		return ErrPeerNameRequired
	}
	if debt.Amount <= 0 {
		return ErrAmountMustBePositive
	}
	if isDateInPast(debt.DueDate, u.now().UTC()) {
		return ErrDueDateInPast
	}

	debt.UserID = existing.UserID
	debt.Status = existing.Status
	debt.CreatedAt = existing.CreatedAt

	return u.repo.Update(ctx, debt)
}

func (u *DebtUsecase) GetByID(ctx context.Context, id string) (*domain.Debt, error) {
	if id == "" {
		return nil, ErrDebtIDRequired
	}
	return u.repo.GetByID(ctx, id)
}

func (u *DebtUsecase) ListByUser(ctx context.Context, userID string, options repository.ListOptions) ([]*domain.Debt, int, error) {
	if userID == "" {
		return nil, 0, ErrUserIDRequired
	}
	return u.repo.ListByUser(ctx, userID, options)
}

func (u *DebtUsecase) ListUpcoming(ctx context.Context, userID string, days int, options repository.ListOptions) ([]*domain.Debt, int, error) {
	if userID == "" {
		return nil, 0, ErrUserIDRequired
	}
	if days <= 0 {
		return nil, 0, errors.New("days must be positive")
	}
	return u.repo.ListUpcoming(ctx, userID, days, options)
}

func (u *DebtUsecase) MarkPaid(ctx context.Context, id string) (*domain.Debt, error) {
	if id == "" {
		return nil, ErrDebtIDRequired
	}
	existing, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.Status == domain.DebtStatusPaid {
		return nil, ErrDebtAlreadyPaid
	}

	return u.repo.MarkPaid(ctx, id)
}

func (u *DebtUsecase) RunOverdueCheck(ctx context.Context) (int64, error) {
	nowUTC := u.now().UTC().Format("2006-01-02")
	return u.repo.SetOverdue(ctx, nowUTC)
}

func (u *DebtUsecase) RunReminderCheck(ctx context.Context) ([]*domain.Debt, error) {
	now := u.now().UTC()
	nowDate := now.Format("2006-01-02")
	nowTimestamp := now.Format("2006-01-02 15:04:05")

	debts, err := u.repo.GetDueForReminder(ctx, nowDate)
	if err != nil {
		return nil, err
	}

	for _, debt := range debts {
		if debt == nil {
			continue
		}
		err = u.repo.UpdateReminder(ctx, debt.ID, nowTimestamp, nowTimestamp)
		if err != nil {
			return nil, err
		}
	}

	return debts, nil
}

func isDateInPast(date time.Time, nowUTC time.Time) bool {
	startOfToday := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC)
	return date.Before(startOfToday)
}

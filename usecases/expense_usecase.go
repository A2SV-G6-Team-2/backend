package usecases

import (
	"context"
	"expense_tracker/domain"
	"expense_tracker/repository"
	"time"
)

// ExpenseUseCase handles expense business logic
type ExpenseUseCase struct {
	expenseRepo repository.ExpenseRepository
}

// NewExpenseUseCase creates a new expense use case
func NewExpenseUseCase(expenseRepo repository.ExpenseRepository) *ExpenseUseCase {
	return &ExpenseUseCase{expenseRepo: expenseRepo}
}

// Create creates a new expense for the given user (ownership enforced by userID)
func (uc *ExpenseUseCase) Create(ctx context.Context, input domain.CreateExpenseInput) (*domain.Expense, error) {
	return uc.expenseRepo.Create(ctx, input)
}

// GetByID returns an expense by ID if it belongs to the user
func (uc *ExpenseUseCase) GetByID(ctx context.Context, id, userID string) (*domain.Expense, error) {
	return uc.expenseRepo.GetByID(ctx, id, userID)
}

// List returns expenses for the user with optional filters (ownership: userID required)
func (uc *ExpenseUseCase) List(ctx context.Context, filter domain.ExpenseFilter) ([]*domain.Expense, int, error) {
	if filter.UserID == "" {
		return nil, 0, nil
	}
	return uc.expenseRepo.List(ctx, filter)
}

// Update updates an expense; ownership enforced (userID)
func (uc *ExpenseUseCase) Update(ctx context.Context, id, userID string, input domain.UpdateExpenseInput) (*domain.Expense, error) {
	return uc.expenseRepo.Update(ctx, id, userID, input)
}

// Delete deletes an expense; ownership enforced (userID)
func (uc *ExpenseUseCase) Delete(ctx context.Context, id, userID string) error {
	return uc.expenseRepo.Delete(ctx, id, userID)
}

// ParseExpenseFilter builds filter from query params (from_date, to_date, category_id)
func ParseExpenseFilter(userID string, fromDate, toDate *time.Time, categoryID *string) domain.ExpenseFilter {
	return domain.ExpenseFilter{
		UserID:     userID,
		CategoryID: categoryID,
		FromDate:   fromDate,
		ToDate:     toDate,
	}
}

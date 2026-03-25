package repository

import (
	"context"
	"time"

	"expense_tracker/domain"

	"github.com/google/uuid"
)

// CategoryTotal holds category name and total for report breakdowns
type CategoryTotal struct {
	CategoryName string
	Total        float64
}

// ExpenseRepository defines persistence for expenses (CRUD + report aggregation)
type ExpenseRepository interface {
	// CRUD (Team 2)
	Create(ctx context.Context, input domain.CreateExpenseInput) (*domain.Expense, error)
	GetByID(ctx context.Context, id, userID string) (*domain.Expense, error)
	List(ctx context.Context, filter domain.ExpenseFilter) ([]*domain.Expense, int, error)
	Update(ctx context.Context, id, userID string, input domain.UpdateExpenseInput) (*domain.Expense, error)
	Delete(ctx context.Context, id, userID string) error
	// Report aggregation (reports usecase)
	SumByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) (float64, error)
	CategoryBreakdownByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]CategoryTotal, error)
}

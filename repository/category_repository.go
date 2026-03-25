package repository

import (
	"context"
	"expense_tracker/domain"
)

type ListOptions struct {
	Limit  int
	Offset int
}

// CategoryRepository defines persistence for categories
type CategoryRepository interface {
	Create(ctx context.Context, input domain.CreateCategoryInput) (*domain.Category, error)
	GetByID(ctx context.Context, id string, userID *string) (*domain.Category, error)
	List(ctx context.Context, userID *string, options ListOptions) ([]*domain.Category, int, error) // nil userID = global only; non-nil = global + user's
	Update(ctx context.Context, id string, userID *string, input domain.UpdateCategoryInput) (*domain.Category, error)
	Delete(ctx context.Context, id string, userID *string) error
}

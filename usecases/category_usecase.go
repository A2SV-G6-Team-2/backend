package usecases

import (
	"context"
	"expense_tracker/domain"
	"expense_tracker/repository"
)

// CategoryUseCase handles category business logic
type CategoryUseCase struct {
	categoryRepo repository.CategoryRepository
}

// NewCategoryUseCase creates a new category use case
func NewCategoryUseCase(categoryRepo repository.CategoryRepository) *CategoryUseCase {
	return &CategoryUseCase{categoryRepo: categoryRepo}
}

// Create creates a new category (global if UserID nil, else user-defined)
func (uc *CategoryUseCase) Create(ctx context.Context, input domain.CreateCategoryInput) (*domain.Category, error) {
	return uc.categoryRepo.Create(ctx, input)
}

// GetByID returns a category by ID if visible to user (global or own)
func (uc *CategoryUseCase) GetByID(ctx context.Context, id string, userID *string) (*domain.Category, error) {
	return uc.categoryRepo.GetByID(ctx, id, userID)
}

// List returns categories: global only if userID nil, else global + user's
func (uc *CategoryUseCase) List(ctx context.Context, userID *string, options repository.ListOptions) ([]*domain.Category, int, error) {
	return uc.categoryRepo.List(ctx, userID, options)
}

// Update updates a category (ownership: only own or global when userID nil)
func (uc *CategoryUseCase) Update(ctx context.Context, id string, userID *string, input domain.UpdateCategoryInput) (*domain.Category, error) {
	return uc.categoryRepo.Update(ctx, id, userID, input)
}

// Delete deletes a category (ownership: only own or global when userID nil)
func (uc *CategoryUseCase) Delete(ctx context.Context, id string, userID *string) error {
	return uc.categoryRepo.Delete(ctx, id, userID)
}

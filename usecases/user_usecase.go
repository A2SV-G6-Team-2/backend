package usecases

import (
	"context"
	"expense_tracker/domain"
	"expense_tracker/repository"

	"github.com/google/uuid"
)

type UserUsecase interface {
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateUserInput) error
}

type UpdateUserInput struct {
	Name            *string
	BudgetingStyle  *string
	DefaultCurrency *string
}

type userUsecase struct {
	userRepo repository.UserRepository
}

func NewUserUsecase(r repository.UserRepository) UserUsecase {
	return &userUsecase{userRepo: r}
}

func (u *userUsecase) GetByID(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	return *user, nil
}

func (u *userUsecase) Update(ctx context.Context, userID uuid.UUID, input UpdateUserInput) error {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if input.Name != nil {
		user.Name = *input.Name
	}
	if input.BudgetingStyle != nil {
		user.BudgetingStyle = *input.BudgetingStyle
	}
	if input.DefaultCurrency != nil {
		user.DefaultCurrency = *input.DefaultCurrency
	}

	return u.userRepo.Update(ctx, user)
}

package usecases

import (
	"context"
	"errors"
	"expense_tracker/repository"
	"time"

	"github.com/google/uuid"
)

// Daily Report Model
type DailyReport struct {
	Date			  string  `json:"date"`
	TotalExpense      float64 `json:"total-expense"`
	TotalLent         float64 `json:"total-lent"`
	TotalBorrowed     float64 `json:"total-borrowed"`
}


type ReportUsecase interface {
	GetWeeklyReport(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) (WeeklyReport, error)
	
	GetDailyReport(ctx context.Context, userID uuid.UUID, date time.Time) (DailyReport, error)
}

var ErrInvalidDateRange = errors.New("end date must be on or after start date")

type WeeklyReport struct {
	StartDate         string                  `json:"start_date"`
	EndDate           string                  `json:"end_date"`
	TotalExpense      float64                 `json:"total_expense"`
	TotalLent         float64                 `json:"total_lent"`
	TotalBorrowed     float64                 `json:"total_borrowed"`
	CategoryBreakdown []WeeklyCategorySummary `json:"category_breakdown"`
}

type WeeklyCategorySummary struct {
	CategoryName string  `json:"category_name"`
	Total        float64 `json:"total"`
}

type reportUsecase struct {
	expenseRepo repository.ExpenseRepository
	debtRepo    repository.DebtReportRepository
}

func NewReportUsecase(expenseRepo repository.ExpenseRepository, debtRepo repository.DebtReportRepository) ReportUsecase {
	return &reportUsecase{expenseRepo: expenseRepo, debtRepo: debtRepo}
}


// Daily Usecase Logic
func (r *reportUsecase) GetDailyReport(ctx context.Context, userID uuid.UUID, date time.Time) (DailyReport, error) {

	totalExpense, err:= r.expenseRepo.SumByDateRange(ctx, userID, date, date)
	if err != nil {
		return DailyReport{}, err
	}

	totalLent, err:= r.debtRepo.SumByDateRangeAndType(ctx, userID, date, date, "lent")
	if err != nil {
		return DailyReport{}, err
	}

	totalBorrowed, err := r.debtRepo.SumByDateRangeAndType(ctx, userID, date, date, "borrowed")
	if err != nil {
		return DailyReport{}, err
	}

	return DailyReport{
		Date:           date.Format("2006-01-02"),
		TotalExpense:   totalExpense,
		TotalLent:      totalLent,
		TotalBorrowed:  totalBorrowed,
	}, nil
}


func (r *reportUsecase) GetWeeklyReport(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) (WeeklyReport, error) {
	if endDate.Before(startDate) {
		return WeeklyReport{}, ErrInvalidDateRange
	}

	totalExpense, err := r.expenseRepo.SumByDateRange(ctx, userID, startDate, endDate)
	if err != nil {
		return WeeklyReport{}, err
	}

	categoryTotals, err := r.expenseRepo.CategoryBreakdownByDateRange(ctx, userID, startDate, endDate)
	if err != nil {
		return WeeklyReport{}, err
	}

	categoryBreakdown := make([]WeeklyCategorySummary, 0, len(categoryTotals))
	for _, item := range categoryTotals {
		categoryBreakdown = append(categoryBreakdown, WeeklyCategorySummary{
			CategoryName: item.CategoryName,
			Total:        item.Total,
		})
	}

	totalLent, err := r.debtRepo.SumByDateRangeAndType(ctx, userID, startDate, endDate, "lent")
	if err != nil {
		return WeeklyReport{}, err
	}

	totalBorrowed, err := r.debtRepo.SumByDateRangeAndType(ctx, userID, startDate, endDate, "borrowed")
	if err != nil {
		return WeeklyReport{}, err
	}

	return WeeklyReport{
		StartDate:         startDate.Format("2006-01-02"),
		EndDate:           endDate.Format("2006-01-02"),
		TotalExpense:      totalExpense,
		TotalLent:         totalLent,
		TotalBorrowed:     totalBorrowed,
		CategoryBreakdown: categoryBreakdown,
	}, nil
}
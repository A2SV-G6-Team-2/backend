package tests

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"expense_tracker/domain"
	deliveryhttp "expense_tracker/delivery/http"
	"expense_tracker/infrastructure/auth"
	"expense_tracker/repository"
	"expense_tracker/usecases"

	"github.com/google/uuid"
)

type fakeUserUsecase struct {
	getByIDFn func(context.Context, uuid.UUID) (domain.User, error)
	updateFn  func(context.Context, uuid.UUID, usecases.UpdateUserInput) error
}

func (f fakeUserUsecase) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	return f.getByIDFn(ctx, id)
}
func (f fakeUserUsecase) Update(ctx context.Context, id uuid.UUID, in usecases.UpdateUserInput) error {
	return f.updateFn(ctx, id, in)
}

type fakeReportUsecase struct {
	dailyFn   func(context.Context, uuid.UUID, time.Time) (usecases.DailyReport, error)
	weeklyFn  func(context.Context, uuid.UUID, time.Time, time.Time) (usecases.WeeklyReport, error)
	monthlyFn func(context.Context, uuid.UUID, int, time.Month) (usecases.MonthlyReport, error)
}

func (f fakeReportUsecase) GetDailyReport(ctx context.Context, id uuid.UUID, date time.Time) (usecases.DailyReport, error) {
	return f.dailyFn(ctx, id, date)
}
func (f fakeReportUsecase) GetWeeklyReport(ctx context.Context, id uuid.UUID, start, end time.Time) (usecases.WeeklyReport, error) {
	return f.weeklyFn(ctx, id, start, end)
}
func (f fakeReportUsecase) GetMonthlyReport(ctx context.Context, id uuid.UUID, year int, month time.Month) (usecases.MonthlyReport, error) {
	return f.monthlyFn(ctx, id, year, month)
}

type fakeExpenseRepo struct {
	createFn func(context.Context, domain.CreateExpenseInput) (*domain.Expense, error)
	getFn    func(context.Context, string, string) (*domain.Expense, error)
	listFn   func(context.Context, domain.ExpenseFilter) ([]*domain.Expense, int, error)
	updateFn func(context.Context, string, string, domain.UpdateExpenseInput) (*domain.Expense, error)
	deleteFn func(context.Context, string, string) error
}

func (f fakeExpenseRepo) Create(ctx context.Context, in domain.CreateExpenseInput) (*domain.Expense, error) {
	return f.createFn(ctx, in)
}
func (f fakeExpenseRepo) GetByID(ctx context.Context, id, userID string) (*domain.Expense, error) {
	return f.getFn(ctx, id, userID)
}
func (f fakeExpenseRepo) List(ctx context.Context, filter domain.ExpenseFilter) ([]*domain.Expense, int, error) {
	return f.listFn(ctx, filter)
}
func (f fakeExpenseRepo) Update(ctx context.Context, id, userID string, in domain.UpdateExpenseInput) (*domain.Expense, error) {
	return f.updateFn(ctx, id, userID, in)
}
func (f fakeExpenseRepo) Delete(ctx context.Context, id, userID string) error {
	return f.deleteFn(ctx, id, userID)
}
func (fakeExpenseRepo) SumByDateRange(context.Context, uuid.UUID, time.Time, time.Time) (float64, error) {
	return 0, nil
}
func (fakeExpenseRepo) CategoryBreakdownByDateRange(context.Context, uuid.UUID, time.Time, time.Time) ([]repository.CategoryTotal, error) {
	return nil, nil
}

type fakeCategoryRepo struct {
	createFn func(context.Context, domain.CreateCategoryInput) (*domain.Category, error)
	getFn    func(context.Context, string, *string) (*domain.Category, error)
	listFn   func(context.Context, *string, repository.ListOptions) ([]*domain.Category, int, error)
	updateFn func(context.Context, string, *string, domain.UpdateCategoryInput) (*domain.Category, error)
	deleteFn func(context.Context, string, *string) error
}

func (f fakeCategoryRepo) Create(ctx context.Context, in domain.CreateCategoryInput) (*domain.Category, error) {
	return f.createFn(ctx, in)
}
func (f fakeCategoryRepo) GetByID(ctx context.Context, id string, userID *string) (*domain.Category, error) {
	return f.getFn(ctx, id, userID)
}
func (f fakeCategoryRepo) List(ctx context.Context, userID *string, opts repository.ListOptions) ([]*domain.Category, int, error) {
	return f.listFn(ctx, userID, opts)
}
func (f fakeCategoryRepo) Update(ctx context.Context, id string, userID *string, in domain.UpdateCategoryInput) (*domain.Category, error) {
	return f.updateFn(ctx, id, userID, in)
}
func (f fakeCategoryRepo) Delete(ctx context.Context, id string, userID *string) error {
	return f.deleteFn(ctx, id, userID)
}

type fakeDebtRepo struct {
	createFn       func(context.Context, *domain.Debt) error
	updateFn       func(context.Context, *domain.Debt) error
	getByIDFn      func(context.Context, string) (*domain.Debt, error)
	listByUserFn   func(context.Context, string, repository.ListOptions) ([]*domain.Debt, int, error)
	listUpcomingFn func(context.Context, string, int, repository.ListOptions) ([]*domain.Debt, int, error)
	markPaidFn     func(context.Context, string) (*domain.Debt, error)
}

func (f fakeDebtRepo) Create(ctx context.Context, debt *domain.Debt) error { return f.createFn(ctx, debt) }
func (f fakeDebtRepo) Update(ctx context.Context, debt *domain.Debt) error { return f.updateFn(ctx, debt) }
func (f fakeDebtRepo) GetByID(ctx context.Context, id string) (*domain.Debt, error) {
	return f.getByIDFn(ctx, id)
}
func (f fakeDebtRepo) ListByUser(ctx context.Context, userID string, opts repository.ListOptions) ([]*domain.Debt, int, error) {
	return f.listByUserFn(ctx, userID, opts)
}
func (f fakeDebtRepo) ListUpcoming(ctx context.Context, userID string, days int, opts repository.ListOptions) ([]*domain.Debt, int, error) {
	return f.listUpcomingFn(ctx, userID, days, opts)
}
func (f fakeDebtRepo) MarkPaid(ctx context.Context, id string) (*domain.Debt, error) { return f.markPaidFn(ctx, id) }
func (fakeDebtRepo) SetOverdue(context.Context, string) (int64, error)                 { return 0, nil }
func (fakeDebtRepo) GetDueForReminder(context.Context, string) ([]*domain.Debt, error) { return nil, nil }
func (fakeDebtRepo) UpdateReminder(context.Context, string, string, string) error      { return nil }

func TestUserHandlers(t *testing.T) {
	jwtSvc := auth.NewJWTService("test-secret")
	userID := uuid.New()
	handler := deliveryhttp.NewUserHandler(fakeUserUsecase{
		getByIDFn: func(_ context.Context, id uuid.UUID) (domain.User, error) {
			return domain.User{UserID: id, Name: "Mike", Email: "mike@example.com"}, nil
		},
		updateFn: func(context.Context, uuid.UUID, usecases.UpdateUserInput) error { return nil },
	}, jwtSvc)

	req := newJSONRequest(t, http.MethodGet, "/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, jwtSvc, userID))
	rec := httptest.NewRecorder()
	handler.GetProfile(rec, req)
	env := decodeEnvelope(t, rec)
	if rec.Code != http.StatusOK || !env.Success {
		t.Fatalf("unexpected profile response: code=%d env=%+v", rec.Code, env)
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/user/update", strings.NewReader("{"))
	updateReq.Header.Set("Authorization", "Bearer "+makeAccessToken(t, jwtSvc, userID))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	handler.UpdateProfile(updateRec, updateReq)
	updateEnv := decodeEnvelope(t, updateRec)
	if updateRec.Code != http.StatusBadRequest || updateEnv.Success {
		t.Fatalf("unexpected update response: code=%d env=%+v", updateRec.Code, updateEnv)
	}
}

func TestExpenseHandlers(t *testing.T) {
	repo := fakeExpenseRepo{
		createFn: func(_ context.Context, in domain.CreateExpenseInput) (*domain.Expense, error) {
			return &domain.Expense{ID: in.ID, UserID: in.UserID, Amount: in.Amount}, nil
		},
		getFn: func(context.Context, string, string) (*domain.Expense, error) { return nil, nil },
		listFn: func(_ context.Context, filter domain.ExpenseFilter) ([]*domain.Expense, int, error) {
			return []*domain.Expense{{ID: "exp-1", UserID: filter.UserID, Amount: 12}}, 1, nil
		},
		updateFn: func(_ context.Context, id, userID string, _ domain.UpdateExpenseInput) (*domain.Expense, error) {
			return &domain.Expense{ID: id, UserID: userID, Amount: 20}, nil
		},
		deleteFn: func(context.Context, string, string) error { return nil },
	}
	handler := deliveryhttp.NewExpenseHandler(usecases.NewExpenseUseCase(repo))
	jwtSvc := auth.NewJWTService("test-secret")
	userID := uuid.New()
	authHeader := "Bearer " + makeAccessToken(t, jwtSvc, userID)

	createReq := newJSONRequest(t, http.MethodPost, "/expenses", map[string]interface{}{
		"amount":       10,
		"expense_date": "2026-01-01",
	})
	createReq.Header.Set("Authorization", authHeader)
	createRec := serveWithExpenseCategoryAuth(jwtSvc, createReq, func(w http.ResponseWriter, r *http.Request) {
		handler.Create(w, r)
	})
	if env := decodeEnvelope(t, createRec); createRec.Code != http.StatusCreated || !env.Success {
		t.Fatalf("unexpected expense create response: code=%d env=%+v", createRec.Code, env)
	}

	listReq := newJSONRequest(t, http.MethodGet, "/expenses?page=1&page_size=10", nil)
	listReq.Header.Set("Authorization", authHeader)
	listRec := serveWithExpenseCategoryAuth(jwtSvc, listReq, func(w http.ResponseWriter, r *http.Request) {
		handler.List(w, r)
	})
	listEnv := decodeEnvelope(t, listRec)
	if listRec.Code != http.StatusOK || !listEnv.Success || listEnv.Meta == nil {
		t.Fatalf("unexpected expense list response: code=%d env=%+v", listRec.Code, listEnv)
	}

	getReq := newJSONRequest(t, http.MethodGet, "/expenses/123e4567-e89b-12d3-a456-426614174000", nil)
	getReq.Header.Set("Authorization", authHeader)
	getRec := serveWithExpenseCategoryAuth(jwtSvc, getReq, func(w http.ResponseWriter, r *http.Request) {
		handler.GetByID(w, r, "123e4567-e89b-12d3-a456-426614174000")
	})
	if env := decodeEnvelope(t, getRec); getRec.Code != http.StatusNotFound || env.Success {
		t.Fatalf("unexpected expense get response: code=%d env=%+v", getRec.Code, env)
	}
}

func TestCategoryHandlers(t *testing.T) {
	repo := fakeCategoryRepo{
		createFn: func(_ context.Context, in domain.CreateCategoryInput) (*domain.Category, error) {
			return &domain.Category{ID: "cat-1", Name: in.Name, UserID: in.UserID}, nil
		},
		getFn: func(_ context.Context, id string, userID *string) (*domain.Category, error) {
			return &domain.Category{ID: id, Name: "Food", UserID: userID}, nil
		},
		listFn: func(_ context.Context, _ *string, _ repository.ListOptions) ([]*domain.Category, int, error) {
			return []*domain.Category{{ID: "cat-1", Name: "Food"}}, 1, nil
		},
		updateFn: func(_ context.Context, id string, _ *string, _ domain.UpdateCategoryInput) (*domain.Category, error) {
			return &domain.Category{ID: id, Name: "Updated"}, nil
		},
		deleteFn: func(context.Context, string, *string) error { return sql.ErrNoRows },
	}
	handler := deliveryhttp.NewCategoryHandler(usecases.NewCategoryUseCase(repo))
	jwtSvc := auth.NewJWTService("test-secret")
	userID := uuid.New()
	authHeader := "Bearer " + makeAccessToken(t, jwtSvc, userID)

	createReq := newJSONRequest(t, http.MethodPost, "/categories", map[string]string{"name": "Food"})
	createReq.Header.Set("Authorization", authHeader)
	createRec := serveWithExpenseCategoryAuth(jwtSvc, createReq, func(w http.ResponseWriter, r *http.Request) {
		handler.Create(w, r)
	})
	if env := decodeEnvelope(t, createRec); createRec.Code != http.StatusCreated || !env.Success {
		t.Fatalf("unexpected category create response: code=%d env=%+v", createRec.Code, env)
	}

	listReq := newJSONRequest(t, http.MethodGet, "/categories?page=1&page_size=10", nil)
	listReq.Header.Set("Authorization", authHeader)
	listRec := serveWithExpenseCategoryAuth(jwtSvc, listReq, func(w http.ResponseWriter, r *http.Request) {
		handler.List(w, r)
	})
	if env := decodeEnvelope(t, listRec); listRec.Code != http.StatusOK || !env.Success || env.Meta == nil {
		t.Fatalf("unexpected category list response: code=%d env=%+v", listRec.Code, env)
	}

	deleteReq := newJSONRequest(t, http.MethodDelete, "/categories/123e4567-e89b-12d3-a456-426614174000", nil)
	deleteReq.Header.Set("Authorization", authHeader)
	deleteRec := serveWithExpenseCategoryAuth(jwtSvc, deleteReq, func(w http.ResponseWriter, r *http.Request) {
		handler.Delete(w, r, "123e4567-e89b-12d3-a456-426614174000")
	})
	if env := decodeEnvelope(t, deleteRec); deleteRec.Code != http.StatusNotFound || env.Success {
		t.Fatalf("unexpected category delete response: code=%d env=%+v", deleteRec.Code, env)
	}
}

func TestDebtHandlersAndRoutes(t *testing.T) {
	userID := uuid.New()
	jwtSvc := auth.NewJWTService("test-secret")
	debtID := "123e4567-e89b-12d3-a456-426614174000"
	repo := fakeDebtRepo{
		createFn: func(context.Context, *domain.Debt) error { return nil },
		updateFn: func(context.Context, *domain.Debt) error { return nil },
		getByIDFn: func(context.Context, string) (*domain.Debt, error) {
			return &domain.Debt{ID: debtID, UserID: userID.String(), Status: domain.DebtStatusPending, DueDate: time.Now().Add(24 * time.Hour)}, nil
		},
		listByUserFn: func(context.Context, string, repository.ListOptions) ([]*domain.Debt, int, error) {
			return []*domain.Debt{{ID: debtID, UserID: userID.String()}}, 1, nil
		},
		listUpcomingFn: func(context.Context, string, int, repository.ListOptions) ([]*domain.Debt, int, error) {
			return []*domain.Debt{{ID: debtID, UserID: userID.String()}}, 1, nil
		},
		markPaidFn: func(context.Context, string) (*domain.Debt, error) {
			return &domain.Debt{ID: debtID, UserID: userID.String(), Status: domain.DebtStatusPaid}, nil
		},
	}
	handler := deliveryhttp.NewDebtHandler(usecases.NewDebtUsecase(repo), jwtSvc)

	createRec := httptest.NewRecorder()
	createReq := newJSONRequest(t, http.MethodPost, "/debts", map[string]interface{}{
		"id":        debtID,
		"type":      "lent",
		"peer_name": "Alex",
		"amount":    10,
		"due_date":  time.Now().Add(24 * time.Hour).Format("2006-01-02"),
	})
	createReq.Header.Set("Authorization", "Bearer "+makeAccessToken(t, jwtSvc, userID))
	handler.CreateDebt(createRec, createReq)
	if env := decodeEnvelope(t, createRec); createRec.Code != http.StatusCreated || !env.Success {
		t.Fatalf("unexpected debt create response: code=%d env=%+v", createRec.Code, env)
	}

	listRec := httptest.NewRecorder()
	listReq := newJSONRequest(t, http.MethodGet, "/debts?page=1&page_size=10", nil)
	listReq.Header.Set("Authorization", "Bearer "+makeAccessToken(t, jwtSvc, userID))
	handler.ListDebts(listRec, listReq)
	if env := decodeEnvelope(t, listRec); listRec.Code != http.StatusOK || !env.Success {
		t.Fatalf("unexpected debt list response: code=%d env=%+v", listRec.Code, env)
	}

	mux := http.NewServeMux()
	deliveryhttp.RegisterDebtRoutes(mux, handler)
	methodRec := httptest.NewRecorder()
	methodReq := newJSONRequest(t, http.MethodDelete, "/debts", nil)
	mux.ServeHTTP(methodRec, methodReq)
	if env := decodeEnvelope(t, methodRec); methodRec.Code != http.StatusMethodNotAllowed || env.Success {
		t.Fatalf("unexpected route response: code=%d env=%+v", methodRec.Code, env)
	}
}

func TestReportHandlersAndMiddleware(t *testing.T) {
	userID := uuid.New()
	jwtSvc := auth.NewJWTService("test-secret")
	handler := deliveryhttp.NewReportHandler(fakeReportUsecase{
		dailyFn: func(context.Context, uuid.UUID, time.Time) (usecases.DailyReport, error) {
			return usecases.DailyReport{Date: "2026-01-01", TotalExpense: 10}, nil
		},
		weeklyFn: func(_ context.Context, _ uuid.UUID, start, end time.Time) (usecases.WeeklyReport, error) {
			if end.Before(start) {
				return usecases.WeeklyReport{}, usecases.ErrInvalidDateRange
			}
			return usecases.WeeklyReport{StartDate: "2026-01-01", EndDate: "2026-01-07"}, nil
		},
		monthlyFn: func(context.Context, uuid.UUID, int, time.Month) (usecases.MonthlyReport, error) {
			return usecases.MonthlyReport{Month: "2026-01"}, nil
		},
	}, jwtSvc)

	dailyRec := httptest.NewRecorder()
	dailyReq := newJSONRequest(t, http.MethodGet, "/reports/daily?date=2026-01-01", nil)
	dailyReq.Header.Set("Authorization", "Bearer "+makeAccessToken(t, jwtSvc, userID))
	handler.GetDailyReport(dailyRec, dailyReq)
	if env := decodeEnvelope(t, dailyRec); dailyRec.Code != http.StatusOK || !env.Success {
		t.Fatalf("unexpected daily response: code=%d env=%+v", dailyRec.Code, env)
	}

	weeklyRec := httptest.NewRecorder()
	weeklyReq := newJSONRequest(t, http.MethodGet, "/reports/weekly?start=2026-01-07&end=2026-01-01", nil)
	weeklyReq.Header.Set("Authorization", "Bearer "+makeAccessToken(t, jwtSvc, userID))
	handler.GetWeeklyReport(weeklyRec, weeklyReq)
	if env := decodeEnvelope(t, weeklyRec); weeklyRec.Code != http.StatusBadRequest || env.Success {
		t.Fatalf("unexpected weekly response: code=%d env=%+v", weeklyRec.Code, env)
	}

	middlewareRec := httptest.NewRecorder()
	middlewareReq := newJSONRequest(t, http.MethodGet, "/expenses", nil)
	deliveryhttp.JWTAuthMiddleware(jwtSvc, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(middlewareRec, middlewareReq)
	if env := decodeEnvelope(t, middlewareRec); middlewareRec.Code != http.StatusUnauthorized || env.Success {
		t.Fatalf("unexpected middleware response: code=%d env=%+v", middlewareRec.Code, env)
	}
}

package http

import (
	"encoding/json"
	"net/http"
	"time"

	"expense_tracker/delivery/apiresponse"
	"expense_tracker/domain"
	"expense_tracker/usecases"

	"github.com/google/uuid"
)

// ExpenseHandler handles expense HTTP endpoints
type ExpenseHandler struct {
	expenseUC *usecases.ExpenseUseCase
}

// NewExpenseHandler creates a new expense handler
func NewExpenseHandler(expenseUC *usecases.ExpenseUseCase) *ExpenseHandler {
	return &ExpenseHandler{expenseUC: expenseUC}
}

// CreateExpenseRequest is the JSON body for POST /expenses
type CreateExpenseRequest struct {
	ID              string  `json:"id"`
	Amount          float64 `json:"amount"`
	CategoryID      *string `json:"category_id,omitempty"`
	IsRecurring     bool    `json:"is_recurring"`
	RecurrenceType  string  `json:"recurrence_type,omitempty"`
	NextDueDate     *string `json:"next_due_date,omitempty"` // YYYY-MM-DD
	ReminderEnabled bool    `json:"reminder_enabled"`
	Note            string  `json:"note,omitempty"`
	ExpenseDate     string  `json:"expense_date"` // YYYY-MM-DD required
}

// UpdateExpenseRequest is the JSON body for PUT /expenses/:id
type UpdateExpenseRequest struct {
	Amount          *float64 `json:"amount,omitempty"`
	CategoryID      *string  `json:"category_id,omitempty"`
	IsRecurring     *bool    `json:"is_recurring,omitempty"`
	RecurrenceType  *string  `json:"recurrence_type,omitempty"`
	NextDueDate     *string  `json:"next_due_date,omitempty"`
	ReminderEnabled *bool    `json:"reminder_enabled,omitempty"`
	Note            *string  `json:"note,omitempty"`
	ExpenseDate     *string  `json:"expense_date,omitempty"`
}

func (h *ExpenseHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
		return
	}
	userID := UserIDFromRequest(r)
	if userID == "" {
		apiresponse.Error(w, http.StatusUnauthorized, "Unauthorized", []string{"authorization required"})
		return
	}

	var req CreateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"invalid request body"})
		return
	}

	// Validation
	if req.Amount <= 0 {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"amount must be positive"})
		return
	}
	expenseDate, err := parseDate(req.ExpenseDate)
	if err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"expense_date is required and must use YYYY-MM-DD"})
		return
	}
	if req.ID != "" && !isValidUUID(req.ID) {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"id must be a valid UUID"})
		return
	}
	if req.CategoryID != nil && *req.CategoryID != "" && !isValidUUID(*req.CategoryID) {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"category_id must be a valid UUID"})
		return
	}
	recType := domain.RecurrenceType(req.RecurrenceType)
	if req.IsRecurring && recType != domain.RecurrenceDaily && recType != domain.RecurrenceWeekly && recType != domain.RecurrenceMonthly {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"recurrence_type must be daily, weekly, or monthly when is_recurring is true"})
		return
	}
	var nextDue *time.Time
	if req.NextDueDate != nil && *req.NextDueDate != "" {
		t, err := parseDate(*req.NextDueDate)
		if err != nil {
			apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"next_due_date must use YYYY-MM-DD"})
			return
		}
		nextDue = &t
	}

	expenseID := req.ID
	if expenseID == "" {
		expenseID = uuid.New().String()
	}

	input := domain.CreateExpenseInput{
		ID:              expenseID,
		UserID:          userID,
		Amount:          req.Amount,
		CategoryID:      req.CategoryID,
		IsRecurring:     req.IsRecurring,
		RecurrenceType:  recType,
		NextDueDate:     nextDue,
		ReminderEnabled: req.ReminderEnabled,
		Note:            req.Note,
		ExpenseDate:     expenseDate,
	}
	expense, err := h.expenseUC.Create(r.Context(), input)
	if err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Expense creation failed", []string{"unable to create expense"})
		return
	}
	apiresponse.Success(w, http.StatusCreated, "Expense created successfully", expense, nil)
}

func (h *ExpenseHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
		return
	}
	userID := UserIDFromRequest(r)
	if userID == "" {
		apiresponse.Error(w, http.StatusUnauthorized, "Unauthorized", []string{"authorization required"})
		return
	}

	pagination, err := apiresponse.ParsePagination(r)
	if err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{err.Error()})
		return
	}

	var fromDate, toDate *time.Time
	if s := r.URL.Query().Get("from_date"); s != "" {
		t, err := parseDate(s)
		if err != nil {
			apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"from_date must use YYYY-MM-DD"})
			return
		}
		fromDate = &t
	}
	if s := r.URL.Query().Get("to_date"); s != "" {
		t, err := parseDate(s)
		if err != nil {
			apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"to_date must use YYYY-MM-DD"})
			return
		}
		toDate = &t
	}
	var categoryID *string
	if s := r.URL.Query().Get("category_id"); s != "" {
		if !isValidUUID(s) {
			apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"category_id must be a valid UUID"})
			return
		}
		categoryID = &s
	}

	filter := usecases.ParseExpenseFilter(userID, fromDate, toDate, categoryID)
	filter.Limit = pagination.PageSize
	filter.Offset = pagination.Offset()

	list, total, err := h.expenseUC.List(r.Context(), filter)
	if err != nil {
		apiresponse.InternalServerError(w)
		return
	}
	apiresponse.PaginatedSuccess(
		w,
		http.StatusOK,
		"Expenses retrieved successfully",
		list,
		apiresponse.NewPaginationMeta(pagination.Page, pagination.PageSize, total),
	)
}

func (h *ExpenseHandler) GetByID(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
		return
	}
	userID := UserIDFromRequest(r)
	if userID == "" {
		apiresponse.Error(w, http.StatusUnauthorized, "Unauthorized", []string{"authorization required"})
		return
	}
	if !isValidUUID(id) {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"invalid expense id"})
		return
	}

	expense, err := h.expenseUC.GetByID(r.Context(), id, userID)
	if err != nil {
		apiresponse.InternalServerError(w)
		return
	}
	if expense == nil {
		apiresponse.Error(w, http.StatusNotFound, "Expense not found", []string{"expense not found"})
		return
	}
	apiresponse.Success(w, http.StatusOK, "Expense retrieved successfully", expense, nil)
}

func (h *ExpenseHandler) Update(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPut {
		apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
		return
	}
	userID := UserIDFromRequest(r)
	if userID == "" {
		apiresponse.Error(w, http.StatusUnauthorized, "Unauthorized", []string{"authorization required"})
		return
	}
	if !isValidUUID(id) {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"invalid expense id"})
		return
	}

	var req UpdateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"invalid request body"})
		return
	}

	input := domain.UpdateExpenseInput{}
	if req.Amount != nil {
		if *req.Amount <= 0 {
			apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"amount must be positive"})
			return
		}
		input.Amount = req.Amount
	}
	input.CategoryID = req.CategoryID
	if req.CategoryID != nil && *req.CategoryID != "" && !isValidUUID(*req.CategoryID) {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"category_id must be a valid UUID"})
		return
	}
	input.IsRecurring = req.IsRecurring
	if req.RecurrenceType != nil {
		rt := domain.RecurrenceType(*req.RecurrenceType)
		input.RecurrenceType = &rt
	}
	if req.NextDueDate != nil {
		t, err := parseDate(*req.NextDueDate)
		if err != nil {
			apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"next_due_date must use YYYY-MM-DD"})
			return
		}
		input.NextDueDate = &t
	}
	input.ReminderEnabled = req.ReminderEnabled
	input.Note = req.Note
	if req.ExpenseDate != nil {
		t, err := parseDate(*req.ExpenseDate)
		if err != nil {
			apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"expense_date must use YYYY-MM-DD"})
			return
		}
		input.ExpenseDate = &t
	}

	expense, err := h.expenseUC.Update(r.Context(), id, userID, input)
	if err != nil {
		apiresponse.InternalServerError(w)
		return
	}
	if expense == nil {
		apiresponse.Error(w, http.StatusNotFound, "Expense not found", []string{"expense not found"})
		return
	}
	apiresponse.Success(w, http.StatusOK, "Expense updated successfully", expense, nil)
}

func (h *ExpenseHandler) Delete(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodDelete {
		apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
		return
	}
	userID := UserIDFromRequest(r)
	if userID == "" {
		apiresponse.Error(w, http.StatusUnauthorized, "Unauthorized", []string{"authorization required"})
		return
	}
	if !isValidUUID(id) {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"invalid expense id"})
		return
	}

	err := h.expenseUC.Delete(r.Context(), id, userID)
	if err != nil {
		if isErrNoRows(err) {
			apiresponse.Error(w, http.StatusNotFound, "Expense not found", []string{"expense not found"})
			return
		}
		apiresponse.InternalServerError(w)
		return
	}
	apiresponse.Success(w, http.StatusOK, "Expense deleted successfully", nil, nil)
}

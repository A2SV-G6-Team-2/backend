package http

import (
	"encoding/json"
	"expense_tracker/delivery/apiresponse"
	"expense_tracker/domain"
	"expense_tracker/infrastructure/auth"
	"expense_tracker/repository"
	"expense_tracker/usecases"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type DebtHandler struct {
	usecase *usecases.DebtUsecase
	jwt     *auth.JWTService
}

func NewDebtHandler(usecase *usecases.DebtUsecase, jwt *auth.JWTService) *DebtHandler {
	return &DebtHandler{usecase: usecase, jwt: jwt}
}

type createDebtRequest struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`
	Type            string  `json:"type"`
	PeerName        string  `json:"peer_name"`
	Amount          float64 `json:"amount"`
	DueDate         string  `json:"due_date"`
	ReminderEnabled bool    `json:"reminder_enabled"`
	Note            *string `json:"note"`
}

type updateDebtRequest struct {
	Type            string  `json:"type"`
	PeerName        string  `json:"peer_name"`
	Amount          float64 `json:"amount"`
	DueDate         string  `json:"due_date"`
	ReminderEnabled bool    `json:"reminder_enabled"`
	Note            *string `json:"note"`
}

func (h *DebtHandler) CreateDebt(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUserID(w, r)
	if !ok {
		return
	}

	var req createDebtRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"invalid request body"})
		return
	}

	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"due_date must use YYYY-MM-DD"})
		return
	}

	debt := &domain.Debt{
		ID:              req.ID,
		UserID:          userID,
		Type:            req.Type,
		PeerName:        req.PeerName,
		Amount:          req.Amount,
		DueDate:         dueDate,
		ReminderEnabled: req.ReminderEnabled,
		Note:            req.Note,
	}

	if err := h.usecase.Create(r.Context(), debt); err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Debt creation failed", []string{err.Error()})
		return
	}

	apiresponse.Success(w, http.StatusCreated, "Debt created successfully", debt, nil)
}

func (h *DebtHandler) UpdateDebt(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUserID(w, r)
	if !ok {
		return
	}

	debtID := extractDebtID(r.URL.Path)
	if debtID == "" {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"missing debt id"})
		return
	}

	existingDebt, err := h.usecase.GetByID(r.Context(), debtID)
	if err != nil {
		apiresponse.Error(w, http.StatusNotFound, "Debt not found", []string{"debt not found"})
		return
	}
	if existingDebt.UserID != userID {
		apiresponse.Error(w, http.StatusForbidden, "Forbidden", []string{"forbidden"})
		return
	}

	var req updateDebtRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"invalid request body"})
		return
	}

	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"due_date must use YYYY-MM-DD"})
		return
	}

	debt := &domain.Debt{
		ID:              debtID,
		Type:            req.Type,
		PeerName:        req.PeerName,
		Amount:          req.Amount,
		DueDate:         dueDate,
		ReminderEnabled: req.ReminderEnabled,
		Note:            req.Note,
	}

	if err := h.usecase.Update(r.Context(), debt); err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Debt update failed", []string{err.Error()})
		return
	}

	apiresponse.Success(w, http.StatusOK, "Debt updated successfully", debt, nil)
}

func (h *DebtHandler) MarkDebtPaid(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUserID(w, r)
	if !ok {
		return
	}

	debtID := extractDebtID(r.URL.Path)
	if debtID == "" {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"missing debt id"})
		return
	}

	existingDebt, err := h.usecase.GetByID(r.Context(), debtID)
	if err != nil {
		apiresponse.Error(w, http.StatusNotFound, "Debt not found", []string{"debt not found"})
		return
	}
	if existingDebt.UserID != userID {
		apiresponse.Error(w, http.StatusForbidden, "Forbidden", []string{"forbidden"})
		return
	}

	debt, err := h.usecase.MarkPaid(r.Context(), debtID)
	if err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Debt payment failed", []string{err.Error()})
		return
	}

	apiresponse.Success(w, http.StatusOK, "Debt marked as paid successfully", debt, nil)
}

func (h *DebtHandler) ListDebts(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUserID(w, r)
	if !ok {
		return
	}

	pagination, err := apiresponse.ParsePagination(r)
	if err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{err.Error()})
		return
	}

	debts, total, err := h.usecase.ListByUser(r.Context(), userID, repository.ListOptions{
		Limit:  pagination.PageSize,
		Offset: pagination.Offset(),
	})
	if err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Unable to retrieve debts", []string{err.Error()})
		return
	}

	apiresponse.PaginatedSuccess(
		w,
		http.StatusOK,
		"Debts retrieved successfully",
		debts,
		apiresponse.NewPaginationMeta(pagination.Page, pagination.PageSize, total),
	)
}

func (h *DebtHandler) ListUpcomingDebts(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUserID(w, r)
	if !ok {
		return
	}

	daysStr := r.URL.Query().Get("days")
	days := 7
	if daysStr != "" {
		parsed, err := parseInt(daysStr)
		if err != nil {
			apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"days must be a valid integer"})
			return
		}
		days = parsed
	}

	pagination, err := apiresponse.ParsePagination(r)
	if err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{err.Error()})
		return
	}

	debts, total, err := h.usecase.ListUpcoming(r.Context(), userID, days, repository.ListOptions{
		Limit:  pagination.PageSize,
		Offset: pagination.Offset(),
	})
	if err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Unable to retrieve upcoming debts", []string{err.Error()})
		return
	}

	apiresponse.PaginatedSuccess(
		w,
		http.StatusOK,
		"Upcoming debts retrieved successfully",
		debts,
		apiresponse.NewPaginationMeta(pagination.Page, pagination.PageSize, total),
	)
}

func extractDebtID(path string) string {
	path = strings.TrimSuffix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}
	if parts[len(parts)-1] == "pay" && len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return parts[len(parts)-1]
}

func parseInt(value string) (int, error) {
	return strconv.Atoi(value)
}

func (h *DebtHandler) authenticatedUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, err := authenticateRequest(r, h.jwt)
	if err != nil {
		writeUnauthorized(w, err)
		return "", false
	}

	return userID.String(), true
}

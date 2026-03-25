package http

import (
	"errors"
	"expense_tracker/delivery/apiresponse"
	"expense_tracker/infrastructure/auth"
	"expense_tracker/usecases"
	"net/http"
	"time"
)

type ReportHandler struct {
	reportUC usecases.ReportUsecase
	jwt      *auth.JWTService
}

func NewReportHandler(uc usecases.ReportUsecase, jwt *auth.JWTService) *ReportHandler {
	return &ReportHandler{reportUC: uc, jwt: jwt}
}

// Daily Handler

func (h *ReportHandler) GetDailyReport(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticateRequest(r, h.jwt)
	if err != nil {
		writeUnauthorized(w, err)
		return
	}

	query := r.URL.Query()
	dateParam := query.Get("date")

	if dateParam == "" {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"date is required"})
		return
	}

	layout := "2006-01-02"

	date, err := time.Parse(layout, dateParam)
	if err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"date must use YYYY-MM-DD"})
		return
	}

	dailyReport, err := h.reportUC.GetDailyReport(r.Context(), userID, date)
	if err != nil {
		apiresponse.InternalServerError(w)
		return
	}

	apiresponse.Success(w, http.StatusOK, "Daily report retrieved successfully", dailyReport, nil)
}

func (h *ReportHandler) GetWeeklyReport(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticateRequest(r, h.jwt)
	if err != nil {
		writeUnauthorized(w, err)
		return
	}

	query := r.URL.Query()
	startDateParam := query.Get("start")
	endDateParam := query.Get("end")
	if startDateParam == "" || endDateParam == "" {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"start and end are required"})
		return
	}

	layout := "2006-01-02"
	startDate, err := time.Parse(layout, startDateParam)
	if err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"start must use YYYY-MM-DD"})
		return
	}
	endDate, err := time.Parse(layout, endDateParam)
	if err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"end must use YYYY-MM-DD"})
		return
	}

	weeklyReport, err := h.reportUC.GetWeeklyReport(r.Context(), userID, startDate, endDate)
	if err != nil {
		if errors.Is(err, usecases.ErrInvalidDateRange) {
			apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{err.Error()})
			return
		}
		apiresponse.InternalServerError(w)
		return
	}

	apiresponse.Success(w, http.StatusOK, "Weekly report retrieved successfully", weeklyReport, nil)
}

// Monthly Handler
func (h *ReportHandler) GetMonthlyReport(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticateRequest(r, h.jwt)
	if err != nil {
		writeUnauthorized(w, err)
		return
	}

	query := r.URL.Query()
	monthParam := query.Get("month")
	if monthParam == "" {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"month is required and must use YYYY-MM"})
		return
	}

	layout := "2006-01"
	parsed, err := time.Parse(layout, monthParam)
	if err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"month must use YYYY-MM"})
		return
	}

	year := parsed.Year()
	month := parsed.Month()

	monthlyReport, err := h.reportUC.GetMonthlyReport(r.Context(), userID, year, month)
	if err != nil {
		apiresponse.InternalServerError(w)
		return
	}

	apiresponse.Success(w, http.StatusOK, "Monthly report retrieved successfully", monthlyReport, nil)
}

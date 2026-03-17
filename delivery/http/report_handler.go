package http

import (
	"errors"
	"expense_tracker/delivery/utils"
	"expense_tracker/infrastructure/auth"
	"expense_tracker/usecases"
	"net/http"
	"strings"
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

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "missing authorization header"})
		return
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	userID, err := h.jwt.Validate(tokenStr)
	if err != nil {
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "invalid token"})
		return
	}

	query := r.URL.Query()
	dateParam := query.Get("date")

	if dateParam == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "date is required"})
		return
	}

	layout := "2006-01-02"

	date, err := time.Parse(layout, dateParam)
	if err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "invalid date format"})
		return
	}

	dailyReport, err := h.reportUC.GetDailyReport(r.Context(), userID, date)
	if err != nil {
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "internal server error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": dailyReport})
}

func (h *ReportHandler) GetWeeklyReport(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "missing authorization header"})
		return
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	userID, err := h.jwt.Validate(tokenStr)
	if err != nil {
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "invalid token"})
		return
	}

	query := r.URL.Query()
	startDateParam := query.Get("start")
	endDateParam := query.Get("end")
	if startDateParam == "" || endDateParam == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "start and end are required"})
		return
	}

	layout := "2006-01-02"
	startDate, err := time.Parse(layout, startDateParam)
	if err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "invalid start date"})
		return
	}
	endDate, err := time.Parse(layout, endDateParam)
	if err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "invalid end date"})
		return
	}

	weeklyReport, err := h.reportUC.GetWeeklyReport(r.Context(), userID, startDate, endDate)
	if err != nil {
		if errors.Is(err, usecases.ErrInvalidDateRange) {
			utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": err.Error()})
			return
		}
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "internal server error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": weeklyReport})
}
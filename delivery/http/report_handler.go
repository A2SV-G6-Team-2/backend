package http

import (
	"errors"
	"expense_tracker/delivery/apiresponse"
	"expense_tracker/infrastructure/auth"
	"expense_tracker/usecases"
	"log"
	"net/http"
	"os"
	"strconv"
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

// Weekly Handler
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

	// Generate an AI insight for the weekly report if the GEMINI_API_KEY env var is set
	insight := ""
	if os.Getenv("GEMINI_API_KEY") != "" {
		// compute previous week range (same length as current)
		days := int(endDate.Sub(startDate).Hours()/24) + 1
		prevEnd := startDate.AddDate(0, 0, -1)
		prevStart := prevEnd.AddDate(0, 0, -(days - 1))
		prevReport, errPrev := h.reportUC.GetWeeklyReport(r.Context(), userID, prevStart, prevEnd)
		if errPrev == nil {
			prompt := buildWeeklyPrompt(weeklyReport, prevReport)
			if s, errGen := generateInsight(r.Context(), prompt); errGen == nil {
				insight = s
				if insight == "" {
					// AI returned empty result
					log.Printf("ai: generateInsight returned empty for weekly prompt: %s", prompt)
					insight = "No insight available"
				}
			} else {
				log.Printf("ai: generateInsight error for weekly: %v", errGen)
				insight = "No insight available"
			}
		} else {
			log.Printf("ai: failed to compute previous weekly report: %v", errPrev)
			insight = "No insight available"
		}
	} else {
		insight = "AI insights disabled (GEMINI_API_KEY not set)"
	}

	// wrap response to include insight (always present)
	type weeklyWithInsight struct {
		usecases.WeeklyReport
		Insight string `json:"insight"`
	}

	apiresponse.Success(w, http.StatusOK, "Weekly report retrieved successfully", weeklyWithInsight{WeeklyReport: weeklyReport, Insight: insight}, nil)
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

	// Generate AI insight for the monthly report if GEMINI_API_KEY is set
	insight := ""
	if os.Getenv("GEMINI_API_KEY") != "" {
		// compute previous month
		prevYear := year
		prevMonth := month - 1
		if prevMonth < 1 {
			prevMonth = 12
			prevYear = year - 1
		}
		prevReport, errPrev := h.reportUC.GetMonthlyReport(r.Context(), userID, prevYear, prevMonth)
		if errPrev == nil {
			prompt := buildMonthlyPrompt(monthlyReport, prevReport)
			if s, errGen := generateInsight(r.Context(), prompt); errGen == nil {
				insight = s
				if insight == "" {
					log.Printf("ai: generateInsight returned empty for monthly prompt: %s", prompt)
					insight = "No insight available"
				}
			} else {
				log.Printf("ai: generateInsight error for monthly: %v", errGen)
				insight = "No insight available"
			}
		} else {
			log.Printf("ai: failed to compute previous monthly report: %v", errPrev)
			insight = "No insight available"
		}
	} else {
		insight = "AI insights disabled (GEMINI_API_KEY not set)"
	}

	type monthlyWithInsight struct {
		usecases.MonthlyReport
		Insight string `json:"insight"`
	}

	apiresponse.Success(w, http.StatusOK, "Monthly report retrieved successfully", monthlyWithInsight{MonthlyReport: monthlyReport, Insight: insight}, nil)
}

// buildWeeklyPrompt composes a concise prompt for the AI based on current and previous weekly reports.
func buildWeeklyPrompt(cur usecases.WeeklyReport, prev usecases.WeeklyReport) string {
	return "Provide a short insight (1-2 sentences) comparing this week's spending to last week's. Include the main habit or change and one suggested action. Current week: total_expense=" + formatFloat(cur.TotalExpense) + ", total_lent=" + formatFloat(cur.TotalLent) + ", total_borrowed=" + formatFloat(cur.TotalBorrowed) + ". Previous week: total_expense=" + formatFloat(prev.TotalExpense) + ", total_lent=" + formatFloat(prev.TotalLent) + ", total_borrowed=" + formatFloat(prev.TotalBorrowed) + "."
}

// buildMonthlyPrompt composes a concise prompt for the AI based on current and previous monthly reports.
func buildMonthlyPrompt(cur usecases.MonthlyReport, prev usecases.MonthlyReport) string {
	return "Provide a short insight (1-2 sentences) comparing this month's spending to last month's and state the current habit based on last month's data. Include one practical suggestion. Current month: total_expense=" + formatFloat(cur.TotalExpense) + ", total_lent=" + formatFloat(cur.TotalLent) + ", total_borrowed=" + formatFloat(cur.TotalBorrowed) + ". Previous month: total_expense=" + formatFloat(prev.TotalExpense) + ", total_lent=" + formatFloat(prev.TotalLent) + ", total_borrowed=" + formatFloat(prev.TotalBorrowed) + "."
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}

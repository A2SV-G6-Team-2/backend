package http

import (
	"net/http"
	"strings"

	"expense_tracker/delivery/apiresponse"
)

// RegisterDebtRoutes registers debt endpoints on mux.
func RegisterDebtRoutes(mux *http.ServeMux, handler *DebtHandler) {
	if mux == nil || handler == nil {
		return
	}

	mux.HandleFunc("/debts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handler.CreateDebt(w, r)
		case http.MethodGet:
			handler.ListDebts(w, r)
		default:
			apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
		}
	})

	mux.HandleFunc("/debts/upcoming", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
			return
		}
		handler.ListUpcomingDebts(w, r)
	})

	mux.HandleFunc("/debts/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/pay") {
			if r.Method != http.MethodPatch {
				apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
				return
			}
			handler.MarkDebtPaid(w, r)
			return
		}

		if r.Method != http.MethodPut {
			apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
			return
		}
		handler.UpdateDebt(w, r)
	})
}

// RegisterExpenseRoutes registers expense endpoints on mux (Team 2).
func RegisterExpenseRoutes(mux *http.ServeMux, handler *ExpenseHandler) {
	if mux == nil || handler == nil {
		return
	}
	mux.HandleFunc("/expenses", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/expenses" {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			handler.List(w, r)
		case http.MethodPost:
			handler.Create(w, r)
		default:
			apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
		}
	})
	mux.HandleFunc("/expenses/", func(w http.ResponseWriter, r *http.Request) {
		id := extractPathID(r.URL.Path, "/expenses/")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			handler.GetByID(w, r, id)
		case http.MethodPut:
			handler.Update(w, r, id)
		case http.MethodDelete:
			handler.Delete(w, r, id)
		default:
			apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
		}
	})
}

// RegisterCategoryRoutes registers category endpoints on mux (Team 2)
func RegisterCategoryRoutes(mux *http.ServeMux, handler *CategoryHandler) {
	if mux == nil || handler == nil {
		return
	}
	mux.HandleFunc("/categories", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/categories" {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			handler.List(w, r)
		case http.MethodPost:
			handler.Create(w, r)
		default:
			apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
		}
	})
	mux.HandleFunc("/categories/", func(w http.ResponseWriter, r *http.Request) {
		id := extractPathID(r.URL.Path, "/categories/")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			handler.GetByID(w, r, id)
		case http.MethodPut:
			handler.Update(w, r, id)
		case http.MethodDelete:
			handler.Delete(w, r, id)
		default:
			apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
		}
	})
}

// extractPathID returns the trailing segment after prefix (e.g. /expenses/uuid -> uuid)
func extractPathID(path, prefix string) string {
	path = strings.TrimSuffix(path, "/")
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	return strings.TrimPrefix(path, prefix)
}

package apiresponse

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Errors  []string    `json:"errors"`
	Meta    interface{} `json:"meta"`
}

type Pagination struct {
	Page        int  `json:"page"`
	PageSize    int  `json:"page_size"`
	TotalItems  int  `json:"total_items"`
	TotalPages  int  `json:"total_pages"`
	HasNext     bool `json:"has_next"`
	HasPrevious bool `json:"has_previous"`
}

type Meta struct {
	Pagination *Pagination `json:"pagination,omitempty"`
}

func Success(w http.ResponseWriter, statusCode int, message string, data interface{}, meta interface{}) {
	writeJSON(w, statusCode, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
		Errors:  nil,
		Meta:    meta,
	})
}

func Error(w http.ResponseWriter, statusCode int, message string, errors []string) {
	if len(errors) == 0 {
		errors = []string{message}
	}

	writeJSON(w, statusCode, APIResponse{
		Success: false,
		Message: message,
		Data:    nil,
		Errors:  errors,
		Meta:    nil,
	})
}

func PaginatedSuccess(w http.ResponseWriter, statusCode int, message string, items interface{}, pagination Pagination) {
	Success(
		w,
		statusCode,
		message,
		map[string]interface{}{"items": items},
		Meta{Pagination: &pagination},
	)
}

func InternalServerError(w http.ResponseWriter) {
	Error(w, http.StatusInternalServerError, "Internal server error", []string{"An unexpected error occurred"})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

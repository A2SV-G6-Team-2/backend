package apiresponse

import (
	"errors"
	"math"
	"net/http"
	"strconv"
)

const (
	DefaultPageSize = 10
	MaxPageSize     = 100
)

type PaginationParams struct {
	Page     int
	PageSize int
}

func ParsePagination(r *http.Request) (PaginationParams, error) {
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return PaginationParams{}, errors.New("page must be an integer greater than or equal to 1")
		}
		page = value
	}

	pageSize := DefaultPageSize
	if raw := r.URL.Query().Get("page_size"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > MaxPageSize {
			return PaginationParams{}, errors.New("page_size must be between 1 and 100")
		}
		pageSize = value
	}

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

func NewPaginationMeta(page, pageSize, totalItems int) Pagination {
	totalPages := 0
	if totalItems > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(pageSize)))
	}

	return Pagination{
		Page:        page,
		PageSize:    pageSize,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}
}

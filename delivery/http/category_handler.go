package http

import (
	"encoding/json"
	"net/http"

	"expense_tracker/delivery/apiresponse"
	"expense_tracker/domain"
	"expense_tracker/repository"
	"expense_tracker/usecases"
)

// CategoryHandler handles category HTTP endpoints
type CategoryHandler struct {
	categoryUC *usecases.CategoryUseCase
}

// NewCategoryHandler creates a new category handler
func NewCategoryHandler(categoryUC *usecases.CategoryUseCase) *CategoryHandler {
	return &CategoryHandler{categoryUC: categoryUC}
}

// CreateCategoryRequest is the JSON body for POST /categories
type CreateCategoryRequest struct {
	Name   string  `json:"name"`
	UserID *string `json:"user_id,omitempty"`
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
		return
	}
	userID := UserIDFromRequest(r)
	// For create: if body has user_id we use it for user-defined; else we can create global (userID nil) or user's (use context)
	var createUserID *string
	if userID != "" {
		createUserID = &userID
	}

	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"invalid request body"})
		return
	}
	if req.Name == "" {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"name is required"})
		return
	}
	// If request specifies user_id, use it (for user-defined category)
	if req.UserID != nil {
		if *req.UserID != "" && !isValidUUID(*req.UserID) {
			apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"user_id must be a valid UUID"})
			return
		}
		createUserID = req.UserID
	}

	input := domain.CreateCategoryInput{Name: req.Name, UserID: createUserID}
	cat, err := h.categoryUC.Create(r.Context(), input)
	if err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Category creation failed", []string{"unable to create category"})
		return
	}
	apiresponse.Success(w, http.StatusCreated, "Category created successfully", cat, nil)
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
		return
	}
	pagination, err := apiresponse.ParsePagination(r)
	if err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{err.Error()})
		return
	}
	var userID *string
	if uid := UserIDFromRequest(r); uid != "" {
		userID = &uid
	}
	list, total, err := h.categoryUC.List(r.Context(), userID, repository.ListOptions{
		Limit:  pagination.PageSize,
		Offset: pagination.Offset(),
	})
	if err != nil {
		apiresponse.InternalServerError(w)
		return
	}
	apiresponse.PaginatedSuccess(
		w,
		http.StatusOK,
		"Categories retrieved successfully",
		list,
		apiresponse.NewPaginationMeta(pagination.Page, pagination.PageSize, total),
	)
}

func (h *CategoryHandler) GetByID(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
		return
	}
	if !isValidUUID(id) {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"invalid category id"})
		return
	}
	var userID *string
	if uid := UserIDFromRequest(r); uid != "" {
		userID = &uid
	}
	cat, err := h.categoryUC.GetByID(r.Context(), id, userID)
	if err != nil {
		apiresponse.InternalServerError(w)
		return
	}
	if cat == nil {
		apiresponse.Error(w, http.StatusNotFound, "Category not found", []string{"category not found"})
		return
	}
	apiresponse.Success(w, http.StatusOK, "Category retrieved successfully", cat, nil)
}

// UpdateCategoryRequest for PUT /categories/:id
type UpdateCategoryRequest struct {
	Name *string `json:"name,omitempty"`
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPut {
		apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
		return
	}
	if !isValidUUID(id) {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"invalid category id"})
		return
	}
	userIDStr := UserIDFromRequest(r)
	var userID *string
	if userIDStr != "" {
		userID = &userIDStr
	}

	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"invalid request body"})
		return
	}
	input := domain.UpdateCategoryInput{Name: req.Name}
	cat, err := h.categoryUC.Update(r.Context(), id, userID, input)
	if err != nil {
		apiresponse.InternalServerError(w)
		return
	}
	if cat == nil {
		apiresponse.Error(w, http.StatusNotFound, "Category not found", []string{"category not found"})
		return
	}
	apiresponse.Success(w, http.StatusOK, "Category updated successfully", cat, nil)
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodDelete {
		apiresponse.Error(w, http.StatusMethodNotAllowed, "Method not allowed", []string{"method not allowed"})
		return
	}
	if !isValidUUID(id) {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"invalid category id"})
		return
	}
	var userID *string
	if uid := UserIDFromRequest(r); uid != "" {
		userID = &uid
	}
	err := h.categoryUC.Delete(r.Context(), id, userID)
	if err != nil {
		if isErrNoRows(err) {
			apiresponse.Error(w, http.StatusNotFound, "Category not found", []string{"category not found"})
			return
		}
		apiresponse.InternalServerError(w)
		return
	}
	apiresponse.Success(w, http.StatusOK, "Category deleted successfully", nil, nil)
}

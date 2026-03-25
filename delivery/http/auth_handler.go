package http

import (
	"encoding/json"
	"expense_tracker/usecases"
	"net/http"

	"expense_tracker/delivery/apiresponse"
)

type AuthHandler struct {
	authUC usecases.AuthUsecase
}

func NewAuthHandler(uc usecases.AuthUsecase) *AuthHandler {
	return &AuthHandler{uc}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input usecases.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"invalid request body"})
		return
	}

	user, err := h.authUC.Register(r.Context(), input)
	if err != nil {
		if err.Error() == "email already used" {
			apiresponse.Error(w, http.StatusConflict, "Registration failed", []string{"email already used"})
			return
		}
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{err.Error()})
		return
	}

	apiresponse.Success(w, http.StatusCreated, "User registered successfully", user, nil)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input usecases.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"invalid request body"})
		return
	}

	resp, err := h.authUC.Login(r.Context(), input)
	if err != nil {
		apiresponse.Error(w, http.StatusUnauthorized, "Authentication failed", []string{"invalid credentials"})
		return
	}

	apiresponse.Success(w, http.StatusOK, "User logged in successfully", resp, nil)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var input usecases.RefreshInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"invalid request body"})
		return
	}

	resp, err := h.authUC.Refresh(r.Context(), input)
	if err != nil {
		switch err.Error() {
		case "refresh token is required":
			apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"refresh_token is required"})
		case "invalid refresh token":
			apiresponse.Error(w, http.StatusUnauthorized, "Refresh failed", []string{"invalid refresh token"})
		default:
			apiresponse.Error(w, http.StatusUnauthorized, "Refresh failed", []string{"unable to refresh token"})
		}
		return
	}

	apiresponse.Success(w, http.StatusOK, "Token refreshed successfully", resp, nil)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var input usecases.LogoutInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"invalid request body"})
		return
	}

	err := h.authUC.Logout(r.Context(), input)
	if err != nil {
		switch err.Error() {
		case "refresh token is required":
			apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"refresh_token is required"})
		case "invalid refresh token":
			apiresponse.Error(w, http.StatusUnauthorized, "Logout failed", []string{"invalid refresh token"})
		default:
			apiresponse.InternalServerError(w)
		}
		return
	}

	apiresponse.Success(w, http.StatusOK, "Logged out successfully", nil, nil)
}

package http

import (
	"encoding/json"
	"net/http"

	"expense_tracker/delivery/apiresponse"
	"expense_tracker/infrastructure/auth"
	"expense_tracker/usecases"
)

type UserHandler struct {
	userUC usecases.UserUsecase
	jwt    *auth.JWTService
}

func NewUserHandler(uc usecases.UserUsecase, jwt *auth.JWTService) *UserHandler {
	return &UserHandler{userUC: uc, jwt: jwt}
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticateRequest(r, h.jwt)
	if err != nil {
		writeUnauthorized(w, err)
		return
	}

	user, err := h.userUC.GetByID(r.Context(), userID)
	if err != nil {
		apiresponse.Error(w, http.StatusNotFound, "User not found", []string{"user not found"})
		return
	}

	apiresponse.Success(w, http.StatusOK, "User fetched successfully", user, nil)
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticateRequest(r, h.jwt)
	if err != nil {
		writeUnauthorized(w, err)
		return
	}

	var input usecases.UpdateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Validation failed", []string{"invalid request body"})
		return
	}

	if err := h.userUC.Update(r.Context(), userID, input); err != nil {
		apiresponse.Error(w, http.StatusBadRequest, "Profile update failed", []string{"unable to update profile"})
		return
	}

	apiresponse.Success(w, http.StatusOK, "Profile updated successfully", nil, nil)
}

package http

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"expense_tracker/delivery/apiresponse"
	"expense_tracker/infrastructure/auth"

	"github.com/google/uuid"
)

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func isErrNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func authenticateRequest(r *http.Request, jwtSvc *auth.JWTService) (uuid.UUID, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return uuid.Nil, errors.New("missing authorization header")
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenStr == "" || tokenStr == authHeader {
		return uuid.Nil, errors.New("invalid authorization header")
	}

	userID, err := jwtSvc.Validate(tokenStr)
	if err != nil {
		return uuid.Nil, errors.New("invalid token")
	}

	return userID, nil
}

func writeUnauthorized(w http.ResponseWriter, err error) {
	apiresponse.Error(w, http.StatusUnauthorized, "Unauthorized", []string{err.Error()})
}

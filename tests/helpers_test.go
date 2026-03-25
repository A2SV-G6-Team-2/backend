package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	deliveryhttp "expense_tracker/delivery/http"
	"expense_tracker/infrastructure/auth"

	"github.com/google/uuid"
)

type apiEnvelope struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    json.RawMessage        `json:"data"`
	Errors  []string               `json:"errors"`
	Meta    map[string]interface{} `json:"meta"`
}

func newJSONRequest(t *testing.T, method, target string, body interface{}) *http.Request {
	t.Helper()

	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(payload)
	}

	req := httptest.NewRequest(method, target, reader)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) apiEnvelope {
	t.Helper()

	var env apiEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode response: %v\nbody=%s", err, rec.Body.String())
	}
	return env
}

func makeAccessToken(t *testing.T, jwtSvc *auth.JWTService, userID uuid.UUID) string {
	t.Helper()

	token, err := jwtSvc.GenerateAccessToken(userID)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	return token
}

func serveWithExpenseCategoryAuth(jwtSvc *auth.JWTService, req *http.Request, next http.HandlerFunc) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	deliveryhttp.JWTAuthMiddleware(jwtSvc, next).ServeHTTP(rec, req)
	return rec
}

func contextBackground() context.Context {
	return context.Background()
}

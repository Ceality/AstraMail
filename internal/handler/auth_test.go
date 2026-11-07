package handler_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"Astramail/internal/handler"
	"Astramail/internal/model"
)

func setupAuthRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(handler.RequireCredentials)
	r.Get("/test-auth", func(w http.ResponseWriter, r *http.Request) {
		creds := handler.Credentials(r.Context())
		writeTestJSON(w, http.StatusOK, creds)
	})

	return r
}

// writeTestJSON 是测试侧的 JSON 输出工具，避免依赖 handler 包的私有函数。
func writeTestJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func TestRequireCredentialsHeaderValidation(t *testing.T) {
	t.Parallel()

	r := setupAuthRouter()

	cases := []struct {
		name         string
		header       string
		expectedCode int
	}{
		{
			name:         "missing_header",
			header:       "",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "wrong_scheme",
			header:       "Bearer sometoken",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "invalid_base64",
			header:       "AstraMail not-valid-base64!",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "invalid_json",
			header:       "AstraMail " + base64.StdEncoding.EncodeToString([]byte("not json")),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "missing_email",
			header:       "AstraMail " + base64.StdEncoding.EncodeToString([]byte(`{"password":"123"}`)),
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test-auth", nil)
		if tc.header != "" {
			req.Header.Set("Authorization", tc.header)
		}

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != tc.expectedCode {
			t.Errorf("[%s] expected status %d, got %d", tc.name, tc.expectedCode, w.Code)
		}
	}
}

func TestRequireCredentialsSuccess(t *testing.T) {
	t.Parallel()

	r := setupAuthRouter()

	rawCreds := []byte(`{"email":"user@example.com","password":"secret"}`)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test-auth", nil)
	req.Header.Set("Authorization", "AstraMail "+base64.StdEncoding.EncodeToString(rawCreds))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var parsed model.AccountCredentials

	err := json.Unmarshal(w.Body.Bytes(), &parsed)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if parsed.Email != "user@example.com" {
		t.Errorf("expected email user@example.com, got %s", parsed.Email)
	}
}

package apperror_test

import (
	"errors"
	"net/http"
	"testing"

	"Astramail/internal/apperror"
)

var errRoot = errors.New("underlying cause")

func TestAppErrorConstructors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		err        *apperror.AppError
		code       apperror.ErrorCode
		statusCode int
	}{
		{
			name:       "bad_request",
			err:        apperror.NewBadRequest("invalid payload", errRoot),
			code:       apperror.CodeBadRequest,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "auth_failed",
			err:        apperror.NewAuthenticationFailed("missing token", errRoot),
			code:       apperror.CodeAuthenticationFailed,
			statusCode: http.StatusUnauthorized,
		},
		{
			name:       "imap_error",
			err:        apperror.NewIMAPError("timeout", errRoot),
			code:       apperror.CodeIMAP,
			statusCode: http.StatusBadGateway,
		},
		{
			name:       "graph_error",
			err:        apperror.NewGraphError("oauth fail", errRoot),
			code:       apperror.CodeGraph,
			statusCode: http.StatusBadGateway,
		},
		{
			name:       "parse_error",
			err:        apperror.NewParseError("bad rfc822", errRoot),
			code:       apperror.CodeParse,
			statusCode: http.StatusUnprocessableEntity,
		},
		{
			name:       "internal_error",
			err:        apperror.NewInternal("disk full", errRoot),
			code:       apperror.CodeInternal,
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		if tc.err.Code != tc.code {
			t.Errorf("[%s] expected code %s, got %s", tc.name, tc.code, tc.err.Code)
		}

		if tc.err.StatusCode != tc.statusCode {
			t.Errorf("[%s] expected status %d, got %d", tc.name, tc.statusCode, tc.err.StatusCode)
		}

		if !errors.Is(tc.err, errRoot) {
			t.Errorf("[%s] errors.Is(tc.err, errRoot) should be true", tc.name)
		}
	}
}

func TestAppErrorFormatting(t *testing.T) {
	t.Parallel()

	withErr := apperror.NewBadRequest("bad field", errRoot)
	if withErr.Error() != "bad field: underlying cause" {
		t.Errorf("unexpected error format: %s", withErr.Error())
	}

	withoutErr := apperror.NewBadRequest("bad field", nil)
	if withoutErr.Error() != "bad field" {
		t.Errorf("unexpected error format: %s", withoutErr.Error())
	}
}

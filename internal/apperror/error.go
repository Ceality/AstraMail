// Package apperror provides application-level error types and code definitions.
package apperror

import (
	"fmt"
	"net/http"
)

// ErrorCode represents application error categories.
type ErrorCode string

// Application error code constants.
const (
	// CodeBadRequest indicates client request validation or parsing failure.
	CodeBadRequest ErrorCode = "BAD_REQUEST"
	// CodeAuthenticationFailed indicates invalid account credentials.
	CodeAuthenticationFailed ErrorCode = "AUTH_FAILED"
	// CodeIMAP indicates an IMAP protocol communication failure.
	CodeIMAP ErrorCode = "IMAP_ERROR"
	// CodeGraph indicates a Microsoft Graph API failure.
	CodeGraph ErrorCode = "GRAPH_ERROR"
	// CodeParse indicates an email parsing failure.
	CodeParse ErrorCode = "PARSE_ERROR"
	// CodeInternal indicates an internal server failure.
	CodeInternal ErrorCode = "INTERNAL_ERROR"
)

// AppError is an application-level error type complying with Go 1.13+ error wrapping.
type AppError struct {
	Code       ErrorCode
	StatusCode int
	Message    string
	Err        error
}

// NewBadRequest creates a 400 Bad Request error.
func NewBadRequest(msg string, err error) *AppError {
	return &AppError{
		Code:       CodeBadRequest,
		StatusCode: http.StatusBadRequest,
		Message:    msg,
		Err:        err,
	}
}

// NewAuthenticationFailed creates a 401 Unauthorized error.
func NewAuthenticationFailed(msg string, err error) *AppError {
	return &AppError{
		Code:       CodeAuthenticationFailed,
		StatusCode: http.StatusUnauthorized,
		Message:    msg,
		Err:        err,
	}
}

// NewIMAPError creates a 502 Bad Gateway IMAP error.
func NewIMAPError(msg string, err error) *AppError {
	return &AppError{
		Code:       CodeIMAP,
		StatusCode: http.StatusBadGateway,
		Message:    "IMAP Error: " + msg,
		Err:        err,
	}
}

// NewGraphError creates a 502 Bad Gateway Microsoft Graph error.
func NewGraphError(msg string, err error) *AppError {
	return &AppError{
		Code:       CodeGraph,
		StatusCode: http.StatusBadGateway,
		Message:    "Graph API Error: " + msg,
		Err:        err,
	}
}

// NewParseError creates a 422 Unprocessable Entity mail parsing error.
func NewParseError(msg string, err error) *AppError {
	return &AppError{
		Code:       CodeParse,
		StatusCode: http.StatusUnprocessableEntity,
		Message:    "Parse Error: " + msg,
		Err:        err,
	}
}

// NewInternal creates a 500 Internal Server Error.
func NewInternal(msg string, err error) *AppError {
	return &AppError{
		Code:       CodeInternal,
		StatusCode: http.StatusInternalServerError,
		Message:    "Internal Error: " + msg,
		Err:        err,
	}
}

// Error returns the human-readable error description.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}

	return e.Message
}

// Unwrap returns the underlying error.
func (e *AppError) Unwrap() error {
	return e.Err
}

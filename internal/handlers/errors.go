// Package handlers implements HTTP request handlers.
package handlers

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents a consistent API error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// WriteError writes a JSON error response with the given status code.
func WriteError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	// encoding error is unlikely and we can't recover
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

// Common error messages for consistency
const (
	ErrBadRequest         = "bad request"
	ErrUnauthorized       = "unauthorized"
	ErrInvalidCredentials = "invalid login or password"
	ErrLoginAlreadyTaken  = "login already taken"
	ErrOrderAlreadyExists = "order already uploaded by another user"
	ErrOrderAlreadyLoaded = "order already uploaded by this user"
	ErrInvalidOrderNumber = "invalid order number format"
	ErrInsufficientFunds  = "insufficient funds"
	ErrInternalServer     = "internal error"
)

// Package errs provides domain error types shared across the application.
// These are returned by services and mapped to HTTP status codes by handlers.
package errs

import "errors"

var (
	// Auth
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrPhoneAlreadyExists = errors.New("phone number already registered")
	ErrAccountDisabled    = errors.New("account is disabled")

	// Lookup
	ErrNotFound = errors.New("resource not found")
	ErrConflict = errors.New("resource already exists")

	// Financial
	ErrInsufficientBalance = errors.New("insufficient wallet balance")
	ErrOptimisticLock      = errors.New("concurrent modification detected, retry")
	ErrIdempotencyConflict = errors.New("idempotency key already used")

	// Authorization
	ErrForbidden = errors.New("action not permitted")

	// Validation
	ErrValidation = errors.New("validation failed")
)

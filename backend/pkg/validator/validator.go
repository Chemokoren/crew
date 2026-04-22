// Package validator provides centralized domain-specific validation for AMY MIS.
// All business rule validation should be routed through this package.
package validator

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// FieldError describes a validation failure for a single field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Result holds the outcome of a validation run.
type Result struct {
	Errors []FieldError
}

// Valid returns true if no validation errors were recorded.
func (r *Result) Valid() bool { return len(r.Errors) == 0 }

// AddError appends a validation error for the given field.
func (r *Result) AddError(field, message string) {
	r.Errors = append(r.Errors, FieldError{Field: field, Message: message})
}

// Error returns a combined error string.
func (r *Result) Error() string {
	if r.Valid() {
		return ""
	}
	msgs := make([]string, len(r.Errors))
	for i, e := range r.Errors {
		msgs[i] = fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return strings.Join(msgs, "; ")
}

// --- Common Validators ---

var (
	phoneRegex    = regexp.MustCompile(`^\+?[1-9]\d{6,14}$`)
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	nationalIDRe  = regexp.MustCompile(`^\d{5,12}$`)
)

// Phone validates a phone number (E.164 format).
func Phone(field, value string, r *Result) {
	if value == "" {
		r.AddError(field, "phone is required")
		return
	}
	if !phoneRegex.MatchString(value) {
		r.AddError(field, "invalid phone number format (expected E.164)")
	}
}

// Email validates an email address (optional — only validated if non-empty).
func Email(field, value string, r *Result) {
	if value == "" {
		return // optional
	}
	if !emailRegex.MatchString(value) {
		r.AddError(field, "invalid email address")
	}
}

// Required validates that a string field is non-empty.
func Required(field, value string, r *Result) {
	if strings.TrimSpace(value) == "" {
		r.AddError(field, fmt.Sprintf("%s is required", field))
	}
}

// MinLength validates minimum string length.
func MinLength(field, value string, min int, r *Result) {
	if len(value) < min {
		r.AddError(field, fmt.Sprintf("%s must be at least %d characters", field, min))
	}
}

// PositiveAmount validates that a monetary amount is positive.
func PositiveAmount(field string, cents int64, r *Result) {
	if cents <= 0 {
		r.AddError(field, "amount must be positive")
	}
}

// NationalID validates a Kenyan national ID number.
func NationalID(field, value string, r *Result) {
	if value == "" {
		r.AddError(field, "national ID is required")
		return
	}
	if !nationalIDRe.MatchString(value) {
		r.AddError(field, "national ID must be 5-12 digits")
	}
}

// FutureDate validates that a date is in the future.
func FutureDate(field string, t time.Time, r *Result) {
	if t.Before(time.Now()) {
		r.AddError(field, fmt.Sprintf("%s must be in the future", field))
	}
}

// DateRange validates that from < to.
func DateRange(fromField, toField string, from, to time.Time, r *Result) {
	if !from.Before(to) {
		r.AddError(toField, fmt.Sprintf("%s must be after %s", toField, fromField))
	}
}

// OneOf validates that a value is in the allowed set.
func OneOf(field, value string, allowed []string, r *Result) {
	for _, a := range allowed {
		if value == a {
			return
		}
	}
	r.AddError(field, fmt.Sprintf("%s must be one of: %s", field, strings.Join(allowed, ", ")))
}

// Package service contains all business logic for AMY MIS.
// Services depend on repository interfaces, never on concrete GORM implementations.
package service

// Domain errors are defined in pkg/errs to avoid import cycles.
// Re-exported here for backward compatibility and convenience.
import "github.com/kibsoft/amy-mis/pkg/errs"

var (
	ErrInvalidCredentials = errs.ErrInvalidCredentials
	ErrPhoneAlreadyExists = errs.ErrPhoneAlreadyExists
	ErrAccountDisabled    = errs.ErrAccountDisabled
	ErrNotFound           = errs.ErrNotFound
	ErrConflict           = errs.ErrConflict
	ErrInsufficientBalance = errs.ErrInsufficientBalance
	ErrOptimisticLock      = errs.ErrOptimisticLock
	ErrIdempotencyConflict = errs.ErrIdempotencyConflict
	ErrForbidden           = errs.ErrForbidden
	ErrValidation          = errs.ErrValidation
)

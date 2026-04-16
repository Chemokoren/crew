package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kibsoft/amy-mis/pkg/pagination"
)

// SuccessResponse sends a success response with data.
func SuccessResponse(c *gin.Context, status int, data interface{}) {
	c.JSON(status, gin.H{
		"success": true,
		"data":    data,
	})
}

// ListResponse sends a paginated list response.
func ListResponse(c *gin.Context, data interface{}, meta pagination.Meta) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"meta":    meta,
	})
}

// ErrorResponse sends an error response.
func ErrorResponse(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

// ValidationErrorResponse sends a validation error with field-level details.
func ValidationErrorResponse(c *gin.Context, details []FieldError) {
	c.JSON(http.StatusBadRequest, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "Validation failed",
			"details": details,
		},
	})
}

// FieldError represents a single field validation error.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// --- Common error helpers ---

func BadRequest(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", message)
}

func Unauthorized(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func Forbidden(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusForbidden, "FORBIDDEN", message)
}

func NotFound(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", message)
}

func Conflict(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusConflict, "CONFLICT", message)
}

func InsufficientBalance(c *gin.Context) {
	ErrorResponse(c, http.StatusUnprocessableEntity, "INSUFFICIENT_BALANCE", "Insufficient wallet balance")
}

func ExternalServiceError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadGateway, "EXTERNAL_SERVICE_ERROR", message)
}

func InternalError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

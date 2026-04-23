package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kibsoft/amy-mis/internal/handler/dto"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/kibsoft/amy-mis/internal/service"
	"github.com/kibsoft/amy-mis/pkg/errs"
	"github.com/kibsoft/amy-mis/pkg/jwt"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authSvc *service.AuthService
}

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	result, err := h.authSvc.Register(c.Request.Context(), service.RegisterInput{
		Phone:      req.Phone,
		Email:      req.Email,
		Password:   req.Password,
		Role:       req.Role,
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		NationalID: req.NationalID,
		CrewRole:   req.CrewRole,
	})
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusCreated, dto.AuthResponse{
		User: dto.UserToResponse(result.User),
		Tokens: dto.TokensDTO{
			AccessToken:  result.Tokens.AccessToken,
			RefreshToken: result.Tokens.RefreshToken,
		},
	})
}

// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	result, err := h.authSvc.Login(c.Request.Context(), service.LoginInput{
		Phone:    req.Phone,
		Password: req.Password,
	})
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, dto.AuthResponse{
		User: dto.UserToResponse(result.User),
		Tokens: dto.TokensDTO{
			AccessToken:  result.Tokens.AccessToken,
			RefreshToken: result.Tokens.RefreshToken,
		},
	})
}

// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	tokens, err := h.authSvc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, dto.TokensDTO{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

// GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		Unauthorized(c, "Authentication required")
		return
	}

	user, err := h.authSvc.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, dto.UserToResponse(user))
}

// GET /api/v1/auth/lookup?phone=+254712345678
// Public endpoint used by USSD gateway to identify registered users.
// Returns minimal user info (no sensitive data).
func (h *AuthHandler) Lookup(c *gin.Context) {
	phone := c.Query("phone")
	if phone == "" {
		BadRequest(c, "phone query parameter is required")
		return
	}

	user, err := h.authSvc.GetUserByPhone(c.Request.Context(), phone)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	// Return only safe, minimal fields needed by USSD gateway
	SuccessResponse(c, http.StatusOK, gin.H{
		"id":             user.ID,
		"phone":          user.Phone,
		"system_role":    user.SystemRole,
		"crew_member_id": user.CrewMemberID,
		"is_active":      user.IsActive,
	})
}

// MapServiceError maps domain errors to HTTP responses. Exported for reuse by other handlers.
func MapServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errs.ErrInvalidCredentials):
		Unauthorized(c, "Invalid phone or password")
	case errors.Is(err, errs.ErrPhoneAlreadyExists):
		Conflict(c, "Phone number already registered")
	case errors.Is(err, errs.ErrAccountDisabled):
		Forbidden(c, "Account is disabled")
	case errors.Is(err, errs.ErrNotFound):
		NotFound(c, "Resource not found")
	case errors.Is(err, errs.ErrConflict):
		Conflict(c, err.Error())
	case errors.Is(err, errs.ErrInsufficientBalance):
		InsufficientBalance(c)
	case errors.Is(err, errs.ErrOptimisticLock):
		ErrorResponse(c, http.StatusConflict, "CONCURRENT_MODIFICATION", "Please retry the operation")
	case errors.Is(err, errs.ErrForbidden):
		Forbidden(c, "Action not permitted")
	case errors.Is(err, errs.ErrValidation):
		BadRequest(c, err.Error())
	default:
		InternalError(c, "An unexpected error occurred")
	}
}

// GetClaimsFromContext is a typed helper for handlers to get JWT claims.
func GetClaimsFromContext(c *gin.Context) *jwt.Claims {
	return middleware.GetClaims(c)
}

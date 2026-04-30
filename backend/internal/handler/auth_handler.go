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
	authSvc  *service.AuthService
	otpSvc   *service.OTPService
}

func NewAuthHandler(authSvc *service.AuthService, otpSvc *service.OTPService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, otpSvc: otpSvc}
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

// POST /api/v1/auth/pin
// Sets or updates the transaction PIN for a user.
func (h *AuthHandler) SetPIN(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
		PIN   string `json:"pin" binding:"required,min=4,max=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.authSvc.SetPIN(c.Request.Context(), req.Phone, req.PIN); err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, gin.H{"message": "PIN set successfully"})
}

// POST /api/v1/auth/pin/verify
// Verifies the transaction PIN for a user.
func (h *AuthHandler) VerifyPIN(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
		PIN   string `json:"pin" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.authSvc.VerifyPIN(c.Request.Context(), req.Phone, req.PIN); err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, gin.H{"valid": true})
}

// POST /api/v1/auth/forgot-password
// Initiates self-service password reset by generating and sending an OTP.
// Supports configurable delivery channels: email (default), sms, or whatsapp.
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Phone   string `json:"phone" binding:"required"`
		Channel string `json:"channel"` // "email" (default), "sms", or "whatsapp"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	genericResponse := gin.H{
		"message":     "If this phone is registered, you will receive an OTP shortly",
		"otp_length":  6,
		"ttl_seconds": 600,
		"channel":     req.Channel,
	}

	// 1. Verify the user exists and is active
	user, err := h.authSvc.GetUserByPhone(c.Request.Context(), req.Phone)
	if err != nil {
		// Don't reveal whether the account exists — generic message
		SuccessResponse(c, http.StatusOK, genericResponse)
		return
	}
	if !user.IsActive {
		SuccessResponse(c, http.StatusOK, genericResponse)
		return
	}

	// 2. Generate and deliver OTP via the messaging engine
	if err := h.otpSvc.GenerateAndSend(c.Request.Context(), req.Phone, user.Email, req.Channel); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// Resolve actual channel used
	channel := req.Channel
	if channel == "" {
		channel = string(h.otpSvc.DefaultChannel())
	}
	genericResponse["channel"] = channel

	SuccessResponse(c, http.StatusOK, genericResponse)
}

// GET /api/v1/auth/otp-channels
// Returns the available OTP delivery channels and the default.
func (h *AuthHandler) OTPChannels(c *gin.Context) {
	channels := h.otpSvc.AvailableChannels()
	channelNames := make([]string, len(channels))
	for i, ch := range channels {
		channelNames[i] = string(ch)
	}
	SuccessResponse(c, http.StatusOK, gin.H{
		"channels":        channelNames,
		"default_channel": string(h.otpSvc.DefaultChannel()),
		"otp_enabled":     h.otpSvc.IsEnabled(),
	})
}

// POST /api/v1/auth/verify-otp
// Verifies the OTP and returns a short-lived reset token.
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
		OTP   string `json:"otp" binding:"required,min=6,max=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	resetToken, err := h.otpSvc.VerifyOTP(c.Request.Context(), req.Phone, req.OTP)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	SuccessResponse(c, http.StatusOK, gin.H{
		"reset_token": resetToken,
		"message":     "OTP verified. Use the reset token to set a new password.",
	})
}

// POST /api/v1/auth/reset-password
// Resets the user's password using a verified reset token (from OTP flow).
func (h *AuthHandler) ResetPasswordOTP(c *gin.Context) {
	var req struct {
		Phone       string `json:"phone" binding:"required"`
		ResetToken  string `json:"reset_token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 1. Validate reset token
	if err := h.otpSvc.ResetPasswordWithToken(c.Request.Context(), req.Phone, req.ResetToken, req.NewPassword); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 2. Look up user and reset password
	user, err := h.authSvc.GetUserByPhone(c.Request.Context(), req.Phone)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	// AdminResetPassword handles hashing internally
	if err := h.authSvc.AdminResetPassword(c.Request.Context(), user.ID, req.NewPassword); err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, gin.H{
		"message": "Password reset successfully. You can now log in with your new password.",
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
	case errors.Is(err, service.ErrLowCreditScore),
		errors.Is(err, service.ErrAmountExceedsTier),
		errors.Is(err, service.ErrTenureExceedsTier),
		errors.Is(err, service.ErrLoanCooldown),
		errors.Is(err, service.ErrActiveLoan),
		errors.Is(err, service.ErrActiveLoanInCat),
		errors.Is(err, service.ErrExposureLimit),
		errors.Is(err, service.ErrCategoryDisabled),
		errors.Is(err, service.ErrInvalidCategory),
		errors.Is(err, service.ErrInvalidStatus):
		BadRequest(c, err.Error())
	default:
		InternalError(c, "An unexpected error occurred")
	}
}

// GetClaimsFromContext is a typed helper for handlers to get JWT claims.
func GetClaimsFromContext(c *gin.Context) *jwt.Claims {
	return middleware.GetClaims(c)
}

package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/external/storage"
	"github.com/kibsoft/amy-mis/internal/handler/dto"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/service"
	"github.com/kibsoft/amy-mis/pkg/errs"
	"github.com/kibsoft/amy-mis/pkg/jwt"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authSvc *service.AuthService
	otpSvc  *service.OTPService
	docSvc  *service.DocumentService // optional — needed for KYC upload
	storage storage.Storage          // optional — needed for KYC upload
}

func NewAuthHandler(authSvc *service.AuthService, otpSvc *service.OTPService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, otpSvc: otpSvc}
}

// WithDocUpload wires document upload dependencies for KYC.
func (h *AuthHandler) WithDocUpload(docSvc *service.DocumentService, store storage.Storage) {
	h.docSvc = docSvc
	h.storage = store
}

// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	result, err := h.authSvc.Register(c.Request.Context(), service.RegisterInput{
		Phone:    req.Phone,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		// Organization fields (for EMPLOYER registration)
		OrganizationName:   req.OrganizationName,
		OrganizationRegNo:  req.OrganizationRegNo,
		OrganizationCounty: req.OrganizationCounty,
		OrganizationPhone:  req.OrganizationPhone,
		IndustryType:       req.IndustryType,
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

	user, crew, restrictions, kycMode, err := h.authSvc.GetEnrichedProfile(c.Request.Context(), claims.UserID)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	resp := dto.EnrichedProfileResponse{
		UserResponse:        dto.UserToResponse(user),
		KYCRestrictions:     restrictions,
		KYCVerificationMode: kycMode,
	}

	if crew != nil {
		resp.CrewProfile = &dto.CrewProfileDTO{
			ID:            crew.ID,
			CrewID:        crew.CrewID,
			FirstName:     crew.FirstName,
			LastName:      crew.LastName,
			FullName:      crew.FullName(),
			Role:          crew.Role,
			JobTypeID:     crew.JobTypeID,
			JobTitle:      crew.JobTitle,
			KYCStatus:     crew.KYCStatus,
			KYCVerifiedAt: crew.KYCVerifiedAt,
		}
	}

	SuccessResponse(c, http.StatusOK, resp)
}

// PUT /api/v1/auth/profile
// Updates the current user's job/specialization on their crew member profile.
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		Unauthorized(c, "Authentication required")
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	crew, err := h.authSvc.UpdateProfile(c.Request.Context(), service.UpdateProfileInput{
		UserID:    claims.UserID,
		Role:      req.Role,
		JobTitle:  req.JobTitle,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, dto.CrewProfileDTO{
		ID:            crew.ID,
		CrewID:        crew.CrewID,
		FirstName:     crew.FirstName,
		LastName:      crew.LastName,
		FullName:      crew.FullName(),
		Role:          crew.Role,
		JobTypeID:     crew.JobTypeID,
		JobTitle:      crew.JobTitle,
		KYCStatus:     crew.KYCStatus,
		KYCVerifiedAt: crew.KYCVerifiedAt,
	})
}

// POST /api/v1/auth/kyc/initiate
// Initiates KYC verification by submitting national ID + serial number.
func (h *AuthHandler) InitiateKYC(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		Unauthorized(c, "Authentication required")
		return
	}

	var req dto.InitiateKYCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	crew, err := h.authSvc.InitiateKYC(c.Request.Context(), service.InitiateKYCInput{
		UserID:       claims.UserID,
		NationalID:   req.NationalID,
		SerialNumber: req.SerialNumber,
	})
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, gin.H{
		"kyc_status":  crew.KYCStatus,
		"crew_id":     crew.CrewID,
		"message":     "KYC verification initiated. Your documents are being reviewed.",
	})
}

// POST /api/v1/auth/kyc/upload
// Uploads front and back photos of a National ID for KYC verification (primary method).
// Expects multipart form with "id_front" and "id_back" file fields, plus "national_id" text field.
func (h *AuthHandler) UploadKYC(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		Unauthorized(c, "Authentication required")
		return
	}

	if h.docSvc == nil || h.storage == nil {
		ErrorResponse(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Document upload is not configured")
		return
	}

	nationalID := c.PostForm("national_id")
	if nationalID == "" {
		BadRequest(c, "national_id is required")
		return
	}

	// Get the crew member from the authenticated user
	user, crew, _, _, err := h.authSvc.GetEnrichedProfile(c.Request.Context(), claims.UserID)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	_ = user
	if crew == nil {
		BadRequest(c, "No crew profile linked to this account")
		return
	}

	// Process each file (front required, back optional but recommended)
	uploadFile := func(fieldName string, docType models.DocumentType) (*models.Document, error) {
		file, err := c.FormFile(fieldName)
		if err != nil {
			return nil, nil // File not provided
		}

		f, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", fieldName, err)
		}
		defer f.Close()

		contentType := file.Header.Get("Content-Type")
		objectName := fmt.Sprintf("kyc/%s/%s/%s", crew.ID.String(), uuid.New().String(), file.Filename)

		path, err := h.storage.UploadFile(c.Request.Context(), objectName, f, file.Size, contentType)
		if err != nil {
			return nil, fmt.Errorf("upload %s: %w", fieldName, err)
		}

		doc, err := h.docSvc.CreateDocument(c.Request.Context(), service.CreateDocumentInput{
			CrewMemberID: &crew.ID,
			DocumentType: docType,
			FileName:     file.Filename,
			FileSize:     file.Size,
			MimeType:     contentType,
			StoragePath:  path,
			UploadedByID: claims.UserID,
		})
		if err != nil {
			return nil, fmt.Errorf("save doc record: %w", err)
		}
		return doc, nil
	}

	frontDoc, err := uploadFile("id_front", models.DocKYCFront)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	if frontDoc == nil {
		BadRequest(c, "id_front file is required")
		return
	}

	backDoc, err := uploadFile("id_back", models.DocKYCBack)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	// Update crew member's national ID and set KYC to pending
	updatedCrew, err := h.authSvc.InitiateKYC(c.Request.Context(), service.InitiateKYCInput{
		UserID:     claims.UserID,
		NationalID: nationalID,
	})
	if err != nil {
		MapServiceError(c, err)
		return
	}

	resp := gin.H{
		"kyc_status":    updatedCrew.KYCStatus,
		"crew_id":       updatedCrew.CrewID,
		"message":       "ID documents uploaded successfully. Your identity is being verified.",
		"front_doc_id":  frontDoc.ID,
	}
	if backDoc != nil {
		resp["back_doc_id"] = backDoc.ID
	}
	SuccessResponse(c, http.StatusOK, resp)
}
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
	// Identity provider / external service errors → 503 Service Unavailable
	case strings.Contains(err.Error(), "identity provider not configured"),
		strings.Contains(err.Error(), "iprs verify"),
		strings.Contains(err.Error(), "iprs verification failed"):
		ErrorResponse(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	// Database constraint violations → 400 with user-friendly message
	case strings.Contains(err.Error(), "violates check constraint"):
		msg := extractConstraintMessage(err.Error())
		BadRequest(c, msg)
	case strings.Contains(err.Error(), "violates unique constraint"):
		msg := extractConstraintMessage(err.Error())
		Conflict(c, msg)
	default:
		// Log the actual error for debugging — this is what shows up in error.log
		slog.Error("unhandled service error",
			slog.String("error", err.Error()),
			slog.String("path", c.Request.URL.Path),
			slog.String("method", c.Request.Method),
		)
		InternalError(c, err.Error())
	}
}

// extractConstraintMessage builds a user-friendly message from a Postgres constraint violation.
func extractConstraintMessage(errMsg string) string {
	// Known constraint → friendly message map
	constraintMap := map[string]string{
		"assignments_work_type_check":        "Invalid work type. Allowed values: SHIFT, DAILY, HOURLY, TASK, PROJECT, BOOKING",
		"assignments_earning_model_check":    "Invalid earning model. Allowed values: FIXED, COMMISSION, HYBRID, HOURLY, DAILY_RATE, PER_TASK, PER_PIECE, SALARY",
		"assignments_status_check":           "Invalid assignment status",
		"assignments_commission_basis_check":  "Invalid commission basis. Allowed values: FARE_TOTAL, TRIP_COUNT, REVENUE",
	}
	for constraint, message := range constraintMap {
		if strings.Contains(errMsg, constraint) {
			return message
		}
	}
	return "Data validation failed. Please check your input values."
}

// GetClaimsFromContext is a typed helper for handlers to get JWT claims.
func GetClaimsFromContext(c *gin.Context) *jwt.Claims {
	return middleware.GetClaims(c)
}

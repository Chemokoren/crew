package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/internal/service"
)

// SupportHandler provides platform support center endpoints.
type SupportHandler struct {
	authSvc    *service.AuthService
	walletSvc  *service.WalletService
	payrollSvc *service.PayrollService
	auditRepo  repository.AuditLogRepository
	otpSvc     *service.OTPService
}

// NewSupportHandler creates a new SupportHandler.
func NewSupportHandler(
	authSvc *service.AuthService,
	walletSvc *service.WalletService,
	payrollSvc *service.PayrollService,
	auditRepo repository.AuditLogRepository,
	otpSvc *service.OTPService,
) *SupportHandler {
	return &SupportHandler{
		authSvc:    authSvc,
		walletSvc:  walletSvc,
		payrollSvc: payrollSvc,
		auditRepo:  auditRepo,
		otpSvc:     otpSvc,
	}
}

// SupportStats godoc
// @Summary Get support center dashboard statistics
// @Tags Support
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/support/stats [get]
func (h *SupportHandler) SupportStats(c *gin.Context) {
	ctx := c.Request.Context()

	// Get system stats
	sysStats, err := h.authSvc.GetSystemStats(ctx)
	if err != nil {
		InternalError(c, "Failed to retrieve system stats")
		return
	}

	// Get failed payroll count
	failedPayrolls := int64(0)
	if h.payrollSvc != nil {
		runs, _, err := h.payrollSvc.ListPayrollRuns(ctx, nil, 1, 1)
		if err == nil {
			// Count runs with FAILED status
			for _, r := range runs {
				if r.Status == models.PayrollFailed {
					failedPayrolls++
				}
			}
		}
	}

	// Aggregate support-specific stats
	stats := gin.H{
		"total_users":               sysStats.TotalUsers,
		"active_users":              sysStats.ActiveUsers,
		"total_crew":                sysStats.TotalCrew,
		"failed_payrolls":           failedPayrolls,
		"total_wallet_balance_cents": 0,
		"total_organizations":       0,
	}

	SuccessResponse(c, http.StatusOK, stats)
}

// UserTimeline godoc
// @Summary Get activity timeline for a specific user
// @Tags Support
// @Produce json
// @Param id path string true "User ID or Crew Member ID"
// @Param action query string false "Filter by action type"
// @Param page query int false "Page number"
// @Param per_page query int false "Items per page"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/support/users/{id}/timeline [get]
func (h *SupportHandler) UserTimeline(c *gin.Context) {
	userID := c.Param("id")
	parsedID, err := uuid.Parse(userID)
	if err != nil {
		BadRequest(c, "Invalid user ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "50"))
	action := c.Query("action")

	// Use the efficient ListByUserID that filters at the database level
	logs, total, err := h.auditRepo.ListByUserID(c.Request.Context(), parsedID, action, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	// Transform to timeline format
	var timeline []gin.H
	for _, log := range logs {
		entry := gin.H{
			"id":          log.ID,
			"action":      log.Action,
			"resource":    log.Resource,
			"resource_id": log.ResourceID,
			"actor_id":    log.UserID,
			"details":     string(log.NewValue),
			"ip_address":  log.IPAddress,
			"created_at":  log.CreatedAt,
		}
		timeline = append(timeline, entry)
	}

	if timeline == nil {
		timeline = []gin.H{}
	}

	ListResponse(c, timeline, buildMeta(page, perPage, total))
}

// ResendOTP godoc
// @Summary Resend verification OTP to a user
// @Tags Support
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/support/users/{id}/resend-otp [post]
func (h *SupportHandler) ResendOTP(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid user ID")
		return
	}

	var req struct {
		Channel string `json:"channel"` // sms, email, whatsapp
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Channel = "sms" // default
	}

	// Look up the user to get their phone/email
	user, err := h.authSvc.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	// Determine destination based on channel
	destination := user.Phone
	if req.Channel == "email" && user.Email != "" {
		destination = user.Email
	}

	if destination == "" {
		BadRequest(c, "No destination available for the selected channel")
		return
	}

	// Send OTP using the OTP service
	channel := req.Channel
	if channel == "" {
		channel = "sms"
	}
	if err := h.otpSvc.GenerateAndSend(c.Request.Context(), user.Phone, user.Email, channel); err != nil {
		InternalError(c, "Failed to send OTP: "+err.Error())
		return
	}

	// Log the action for audit trail
	claims := middleware.GetClaims(c)
	actorID := "system"
	if claims != nil {
		actorID = claims.UserID.String()
	}

	// Create audit log entry
	details, _ := json.Marshal(gin.H{
		"channel":  channel,
		"user_id":  userID.String(),
		"sent_by":  actorID,
	})
	auditLog := &models.AuditLog{
		Action:   "RESEND_OTP",
		Resource: "user",
		NewValue: details,
	}
	if claims != nil {
		auditLog.UserID = &claims.UserID
	}
	auditLog.ResourceID = &userID
	auditLog.IPAddress = c.ClientIP()
	auditLog.UserAgent = c.GetHeader("User-Agent")
	_ = h.auditRepo.Create(c.Request.Context(), auditLog)

	// Mask the destination for privacy
	masked := maskDestination(destination)

	SuccessResponse(c, http.StatusOK, gin.H{
		"message":     "Verification code sent successfully",
		"channel":     channel,
		"destination": masked,
		"sent_at":     time.Now().UTC(),
		"sent_by":     actorID,
	})
}

// SearchUsers godoc
// @Summary Search users across multiple fields for support lookup
// @Tags Support
// @Produce json
// @Param q query string true "Search query (phone, email, name, ID)"
// @Param page query int false "Page number"
// @Param per_page query int false "Items per page"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/support/search [get]
func (h *SupportHandler) SearchUsers(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		BadRequest(c, "Search query is required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	// Try exact lookups first for better performance

	// If query looks like a UUID, try direct lookup
	if _, err := uuid.Parse(query); err == nil {
		uid, _ := uuid.Parse(query)
		user, err := h.authSvc.GetUserByID(c.Request.Context(), uid)
		if err == nil {
			ListResponse(c, []interface{}{user}, buildMeta(1, 1, 1))
			return
		}
	}

	// If query looks like a phone number, try phone lookup
	if strings.HasPrefix(query, "+") || strings.HasPrefix(query, "0") || strings.HasPrefix(query, "254") {
		user, err := h.authSvc.GetUserByPhone(c.Request.Context(), query)
		if err == nil {
			ListResponse(c, []interface{}{user}, buildMeta(1, 1, 1))
			return
		}
	}

	// Fallback: server-side filtered search across phone/email
	users, total, err := h.authSvc.ListUsers(c.Request.Context(), page, perPage, query)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	var result []interface{}
	for _, u := range users {
		result = append(result, u)
	}
	if result == nil {
		result = []interface{}{}
	}

	ListResponse(c, result, buildMeta(page, perPage, total))
}

// maskDestination masks a phone number or email for privacy
func maskDestination(dest string) string {
	if strings.Contains(dest, "@") {
		// Email: show prefix chars + domain
		parts := strings.SplitN(dest, "@", 2)
		if len(parts[0]) >= 2 {
			return parts[0][:2] + "***@" + parts[1]
		}
		if len(parts[0]) >= 1 {
			return parts[0][:1] + "***@" + parts[1]
		}
		return "***@" + parts[1]
	}
	// Phone: show last 4 digits
	if len(dest) > 4 {
		return strings.Repeat("*", len(dest)-4) + dest[len(dest)-4:]
	}
	return "****"
}

package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/internal/service"
)

// --- Credit Handler ---

type CreditHandler struct {
	creditSvc service.CreditService
}

func NewCreditHandler(svc service.CreditService) *CreditHandler {
	return &CreditHandler{creditSvc: svc}
}

// GetScore godoc
// @Summary GetScore
// @Description GetScore CreditHandler
// @Tags Credit
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/credit/{crew_member_id} [get]
func (h *CreditHandler) GetScore(c *gin.Context) {
	crewID, err := uuid.Parse(c.Param("crew_member_id"))
	if err != nil {
		BadRequest(c, "Invalid crew member ID")
		return
	}

	// Enforce wallet/financial access rules
	if denied := enforceWalletAccess(c, crewID); denied {
		return
	}

	score, err := h.creditSvc.GetScore(c.Request.Context(), crewID)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, score)
}

// CalculateScore godoc
// @Summary CalculateScore
// @Description CalculateScore CreditHandler
// @Tags Credit
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/credit/{crew_member_id}/calculate [post]
func (h *CreditHandler) CalculateScore(c *gin.Context) {
	crewID, err := uuid.Parse(c.Param("crew_member_id"))
	if err != nil {
		BadRequest(c, "Invalid crew member ID")
		return
	}

	// Only admins or background jobs should calculate scores dynamically
	claims := middleware.GetClaims(c)
	if claims == nil || claims.SystemRole != "SYSTEM_ADMIN" {
		Forbidden(c, "Insufficient permissions to calculate credit scores")
		return
	}

	score, err := h.creditSvc.CalculateScore(c.Request.Context(), crewID)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, score)
}

// GetDetailedScore godoc
// @Summary Get detailed credit score with factor breakdown
// @Description Returns the full credit score with individual factor contributions, suggestions, and feature data
// @Tags Credit
// @Accept json
// @Produce json
// @Param crew_member_id path string true "Crew Member ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/credit/{crew_member_id}/detailed [get]
func (h *CreditHandler) GetDetailedScore(c *gin.Context) {
	crewID, err := uuid.Parse(c.Param("crew_member_id"))
	if err != nil {
		BadRequest(c, "Invalid crew member ID")
		return
	}

	if denied := enforceWalletAccess(c, crewID); denied {
		return
	}

	result, err := h.creditSvc.GetDetailedScore(c.Request.Context(), crewID)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, result)
}

// GetScoreHistory godoc
// @Summary Get score history for a crew member
// @Description Returns historical score computations for trajectory analysis
// @Tags Credit
// @Produce json
// @Param crew_member_id path string true "Crew Member ID"
// @Param limit query int false "Number of records (default 30, max 100)"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/credit/{crew_member_id}/history [get]
func (h *CreditHandler) GetScoreHistory(c *gin.Context) {
	crewID, err := uuid.Parse(c.Param("crew_member_id"))
	if err != nil {
		BadRequest(c, "Invalid crew member ID")
		return
	}

	if denied := enforceWalletAccess(c, crewID); denied {
		return
	}

	limit := 30
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	history, err := h.creditSvc.GetScoreHistory(c.Request.Context(), crewID, limit)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, history)
}

// --- Loan Handler ---

type LoanHandler struct {
	loanSvc service.LoanService
}

func NewLoanHandler(svc service.LoanService) *LoanHandler {
	return &LoanHandler{loanSvc: svc}
}

// GetTier godoc
// @Summary Get loan tier for a crew member
// @Description Returns the user's qualified loan tier (max amount, rate, tenure) based on credit score
// @Tags Loan
// @Accept json
// @Produce json
// @Param crew_member_id path string true "Crew Member ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/loans/tier/{crew_member_id} [get]
func (h *LoanHandler) GetTier(c *gin.Context) {
	crewID, err := uuid.Parse(c.Param("crew_member_id"))
	if err != nil {
		BadRequest(c, "Invalid crew member ID")
		return
	}

	if denied := enforceWalletAccess(c, crewID); denied {
		return
	}

	tier, score, err := h.loanSvc.GetLoanTier(c.Request.Context(), crewID)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, gin.H{
		"score":          score,
		"grade":          tier.Grade,
		"max_loan_kes":   tier.FormatMaxLoanKES(),
		"interest_rate":  tier.FormatInterestPercent(),
		"max_tenure_days": tier.MaxTenureDays,
		"cooldown_days":  tier.CooldownDays,
		"description":    tier.Description,
	})
}

// Apply godoc
// @Summary Apply
// @Description Apply LoanHandler
// @Tags Loan
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/loans [post]
func (h *LoanHandler) Apply(c *gin.Context) {
	var req struct {
		CrewMemberID uuid.UUID `json:"crew_member_id" binding:"required"`
		AmountCents  int64     `json:"amount_cents" binding:"required"`
		TenureDays   int       `json:"tenure_days" binding:"required"`
		Purpose      string    `json:"purpose"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// Crew members can only apply for themselves
	if denied := enforceWalletAccess(c, req.CrewMemberID); denied {
		return
	}

	loan, err := h.loanSvc.ApplyForLoan(c.Request.Context(), req.CrewMemberID, req.AmountCents, req.TenureDays)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, loan)
}

// Approve godoc
// @Summary Approve
// @Description Approve LoanHandler
// @Tags Loan
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/loans/{id}/approve [post]
func (h *LoanHandler) Approve(c *gin.Context) {
	loanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid loan ID")
		return
	}

	var req struct {
		ApprovedAmountCents int64   `json:"approved_amount_cents" binding:"required"`
		InterestRate        float64 `json:"interest_rate" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		Unauthorized(c, "Authentication required")
		return
	}

	loan, err := h.loanSvc.ApproveLoan(c.Request.Context(), loanID, claims.UserID, req.ApprovedAmountCents, req.InterestRate)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, loan)
}

// Reject godoc
// @Summary Reject
// @Description Reject LoanHandler
// @Tags Loan
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/loans/{id}/reject [post]
func (h *LoanHandler) Reject(c *gin.Context) {
	loanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid loan ID")
		return
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		Unauthorized(c, "Authentication required")
		return
	}

	loan, err := h.loanSvc.RejectLoan(c.Request.Context(), loanID)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, loan)
}

// Disburse godoc
// @Summary Disburse
// @Description Disburse LoanHandler
// @Tags Loan
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/loans/{id}/disburse [post]
func (h *LoanHandler) Disburse(c *gin.Context) {
	loanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid loan ID")
		return
	}

	loan, err := h.loanSvc.DisburseLoan(c.Request.Context(), loanID)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, loan)
}

// Repay godoc
// @Summary Repay a loan
// @Description Process a loan repayment — debits wallet, tracks on-time vs late
// @Tags Loan
// @Accept json
// @Produce json
// @Param id path string true "Loan ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/loans/{id}/repay [post]
func (h *LoanHandler) Repay(c *gin.Context) {
	loanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid loan ID")
		return
	}

	var req struct {
		AmountCents int64 `json:"amount_cents" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	loan, err := h.loanSvc.RepayLoan(c.Request.Context(), loanID, req.AmountCents)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, loan)
}

// List godoc
// @Summary List
// @Description List LoanHandler
// @Tags Loan
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/loans [get]
func (h *LoanHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	
	crewIDStr := c.Query("crew_member_id")
	var crewID *uuid.UUID
	if crewIDStr != "" {
		id, err := uuid.Parse(crewIDStr)
		if err == nil {
			crewID = &id
		}
	}

	var filter repository.LoanApplicationFilter
	if crewID != nil {
		filter.CrewMemberID = crewID
	}

	loans, total, err := h.loanSvc.ListLoans(c.Request.Context(), filter, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	ListResponse(c, loans, buildMeta(page, perPage, total))
}

// --- Insurance Handler ---

type InsuranceHandler struct {
	insuranceSvc service.InsuranceService
}

func NewInsuranceHandler(svc service.InsuranceService) *InsuranceHandler {
	return &InsuranceHandler{insuranceSvc: svc}
}

// Create godoc
// @Summary Create
// @Description Create InsuranceHandler
// @Tags Insurance
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/insurance [post]
func (h *InsuranceHandler) Create(c *gin.Context) {
	var req struct {
		CrewMemberID uuid.UUID `json:"crew_member_id" binding:"required"`
		Provider     string    `json:"provider" binding:"required"`
		PolicyType   string    `json:"policy_type" binding:"required"`
		Frequency    string    `json:"frequency" binding:"required"`
		PremiumCents int64     `json:"premium_cents" binding:"required"`
		StartDate    string    `json:"start_date" binding:"required"`
		EndDate      string    `json:"end_date" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	endDate, _ := time.Parse("2006-01-02", req.EndDate)

	policy, err := h.insuranceSvc.CreatePolicy(c.Request.Context(), req.CrewMemberID, req.Provider, req.PolicyType, req.Frequency, req.PremiumCents, startDate, endDate)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, policy)
}

// Lapse godoc
// @Summary Lapse
// @Description Lapse InsuranceHandler
// @Tags Insurance
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/insurance/{id}/lapse [post]
func (h *InsuranceHandler) Lapse(c *gin.Context) {
	policyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid policy ID")
		return
	}

	err = h.insuranceSvc.MarkPolicyLapsed(c.Request.Context(), policyID)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, gin.H{"message": "Policy marked as lapsed"})
}

// List godoc
// @Summary List
// @Description List InsuranceHandler
// @Tags Insurance
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/insurance [get]
func (h *InsuranceHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	
	crewIDStr := c.Query("crew_member_id")
	var crewID *uuid.UUID
	if crewIDStr != "" {
		id, err := uuid.Parse(crewIDStr)
		if err == nil {
			crewID = &id
		}
	}

	var filter repository.InsurancePolicyFilter
	if crewID != nil {
		filter.CrewMemberID = crewID
	}

	policies, total, err := h.insuranceSvc.ListPolicies(c.Request.Context(), filter, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	ListResponse(c, policies, buildMeta(page, perPage, total))
}

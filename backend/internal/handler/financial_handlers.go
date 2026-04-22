package handler

import (
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

// --- Loan Handler ---

type LoanHandler struct {
	loanSvc service.LoanService
}

func NewLoanHandler(svc service.LoanService) *LoanHandler {
	return &LoanHandler{loanSvc: svc}
}

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

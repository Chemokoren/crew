package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/handler/dto"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/internal/service"
	"github.com/kibsoft/amy-mis/pkg/pagination"
)

// AssignmentHandler handles shift assignment endpoints.
type AssignmentHandler struct {
	assignmentSvc *service.AssignmentService
}

func NewAssignmentHandler(svc *service.AssignmentService) *AssignmentHandler {
	return &AssignmentHandler{assignmentSvc: svc}
}

// POST /api/v1/assignments
func (h *AssignmentHandler) Create(c *gin.Context) {
	var req dto.CreateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	shiftDate, err := time.Parse("2006-01-02", req.ShiftDate)
	if err != nil {
		BadRequest(c, "shift_date must be YYYY-MM-DD format")
		return
	}

	shiftStart, err := time.Parse(time.RFC3339, req.ShiftStart)
	if err != nil {
		BadRequest(c, "shift_start must be RFC3339 format")
		return
	}

	claims := middleware.GetClaims(c)

	result, err := h.assignmentSvc.CreateAssignment(c.Request.Context(), service.CreateAssignmentInput{
		CrewMemberID:     req.CrewMemberID,
		VehicleID:        req.VehicleID,
		SaccoID:          req.SaccoID,
		RouteID:          req.RouteID,
		ShiftDate:        shiftDate,
		ShiftStart:       shiftStart,
		EarningModel:     req.EarningModel,
		FixedAmountCents: req.FixedAmountCents,
		CommissionRate:   req.CommissionRate,
		HybridBaseCents:  req.HybridBaseCents,
		CommissionBasis:  req.CommissionBasis,
		Notes:            req.Notes,
		CreatedByID:      claims.UserID,
	})
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusCreated, dto.AssignmentToResponse(result))
}

// POST /api/v1/assignments/:id/complete
func (h *AssignmentHandler) Complete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid assignment ID")
		return
	}

	var req dto.CompleteAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	earning, err := h.assignmentSvc.CompleteAssignment(c.Request.Context(), id, req.TotalRevenueCents)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, gin.H{
		"earning_id":   earning.ID,
		"amount_cents": earning.AmountCents,
		"earning_type": earning.EarningType,
		"message":      "Assignment completed and earnings credited to wallet",
	})
}

// GET /api/v1/assignments/:id
func (h *AssignmentHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid assignment ID")
		return
	}

	assignment, err := h.assignmentSvc.GetAssignment(c.Request.Context(), id)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, dto.AssignmentToResponse(assignment))
}

// GET /api/v1/assignments
func (h *AssignmentHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	filter := repository.AssignmentFilter{}
	if saccoID := c.Query("sacco_id"); saccoID != "" {
		id, _ := uuid.Parse(saccoID)
		filter.SaccoID = &id
	}
	if crewID := c.Query("crew_member_id"); crewID != "" {
		id, _ := uuid.Parse(crewID)
		filter.CrewMemberID = &id
	}
	if status := c.Query("status"); status != "" {
		filter.Status = status
	}
	if dateStr := c.Query("shift_date"); dateStr != "" {
		if d, err := time.Parse("2006-01-02", dateStr); err == nil {
			filter.ShiftDate = &d
		}
	}

	assignments, total, err := h.assignmentSvc.ListAssignments(c.Request.Context(), filter, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	ListResponse(c, dto.AssignmentListToResponse(assignments), buildMeta(page, perPage, total))
}

// --- Wallet Handler ---

// WalletHandler handles wallet and transaction endpoints.
type WalletHandler struct {
	walletSvc *service.WalletService
}

func NewWalletHandler(svc *service.WalletService) *WalletHandler {
	return &WalletHandler{walletSvc: svc}
}

// GET /api/v1/wallets/:crew_member_id
func (h *WalletHandler) GetBalance(c *gin.Context) {
	crewMemberID, err := uuid.Parse(c.Param("crew_member_id"))
	if err != nil {
		BadRequest(c, "Invalid crew member ID")
		return
	}

	wallet, err := h.walletSvc.GetBalance(c.Request.Context(), crewMemberID)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, dto.WalletResponse{
		ID:                 wallet.ID,
		CrewMemberID:       wallet.CrewMemberID,
		BalanceCents:       wallet.BalanceCents,
		BalanceFormatted:   formatKES(wallet.BalanceCents),
		TotalCreditedCents: wallet.TotalCreditedCents,
		TotalDebitedCents:  wallet.TotalDebitedCents,
		Currency:           wallet.Currency,
		IsActive:           wallet.IsActive,
		LastPayoutAt:       wallet.LastPayoutAt,
	})
}

// POST /api/v1/wallets/credit
func (h *WalletHandler) Credit(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		BadRequest(c, "Idempotency-Key header is required for financial operations")
		return
	}

	var req dto.CreditWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	tx, err := h.walletSvc.Credit(c.Request.Context(), service.CreditInput{
		CrewMemberID:   req.CrewMemberID,
		AmountCents:    req.AmountCents,
		Category:       req.Category,
		IdempotencyKey: idempotencyKey,
		Reference:      req.Reference,
		Description:    req.Description,
	})
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusCreated, dto.WalletTxToResponse(tx))
}

// POST /api/v1/wallets/debit
func (h *WalletHandler) Debit(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		BadRequest(c, "Idempotency-Key header is required for financial operations")
		return
	}

	var req dto.DebitWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	tx, err := h.walletSvc.Debit(c.Request.Context(), service.DebitInput{
		CrewMemberID:   req.CrewMemberID,
		AmountCents:    req.AmountCents,
		Category:       req.Category,
		IdempotencyKey: idempotencyKey,
		Reference:      req.Reference,
		Description:    req.Description,
	})
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusCreated, dto.WalletTxToResponse(tx))
}

// GET /api/v1/wallets/:crew_member_id/transactions
func (h *WalletHandler) ListTransactions(c *gin.Context) {
	crewMemberID, err := uuid.Parse(c.Param("crew_member_id"))
	if err != nil {
		BadRequest(c, "Invalid crew member ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	filter := repository.TxFilter{
		Category:        c.Query("category"),
		TransactionType: c.Query("transaction_type"),
		Status:          c.Query("status"),
	}

	txs, total, err := h.walletSvc.GetTransactions(c.Request.Context(), crewMemberID, filter, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	ListResponse(c, dto.WalletTxListToResponse(txs), buildMeta(page, perPage, total))
}

// --- Crew Handler ---

// CrewHandler handles crew member endpoints.
type CrewHandler struct {
	crewSvc *service.CrewService
}

func NewCrewHandler(svc *service.CrewService) *CrewHandler {
	return &CrewHandler{crewSvc: svc}
}

// POST /api/v1/crew
func (h *CrewHandler) Create(c *gin.Context) {
	var req dto.CreateCrewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	crew, err := h.crewSvc.CreateCrewMember(c.Request.Context(), service.CreateCrewInput{
		NationalID: req.NationalID,
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		Role:       req.Role,
	})
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusCreated, dto.CrewToResponse(crew))
}

// GET /api/v1/crew/:id
func (h *CrewHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid crew member ID")
		return
	}

	crew, err := h.crewSvc.GetCrewMember(c.Request.Context(), id)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, dto.CrewToResponse(crew))
}

// PUT /api/v1/crew/:id/kyc
func (h *CrewHandler) UpdateKYC(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid crew member ID")
		return
	}

	var req dto.UpdateKYCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	crew, err := h.crewSvc.UpdateKYCStatus(c.Request.Context(), service.UpdateKYCInput{
		CrewMemberID: id,
		Status:       req.KYCStatus,
	})
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, dto.CrewToResponse(crew))
}

// GET /api/v1/crew
func (h *CrewHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	filter := repository.CrewFilter{
		Role:      c.Query("role"),
		KYCStatus: c.Query("kyc_status"),
		Search:    c.Query("search"),
	}
	if saccoID := c.Query("sacco_id"); saccoID != "" {
		id, _ := uuid.Parse(saccoID)
		filter.SaccoID = &id
	}

	members, total, err := h.crewSvc.ListCrewMembers(c.Request.Context(), filter, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	ListResponse(c, dto.CrewListToResponse(members), buildMeta(page, perPage, total))
}

// DELETE /api/v1/crew/:id
func (h *CrewHandler) Deactivate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid crew member ID")
		return
	}

	if err := h.crewSvc.DeactivateCrewMember(c.Request.Context(), id); err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, gin.H{"message": "Crew member deactivated"})
}

// --- Helpers ---

func buildMeta(page, perPage int, total int64) pagination.Meta {
	totalInt := int(total)
	totalPages := totalInt / perPage
	if totalInt%perPage != 0 {
		totalPages++
	}
	return pagination.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      totalInt,
		TotalPages: totalPages,
	}
}

func formatKES(cents int64) string {
	return fmt.Sprintf("KES %.2f", float64(cents)/100)
}

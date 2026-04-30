package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/external/payment"
	"github.com/kibsoft/amy-mis/internal/handler/dto"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/internal/service"
	"github.com/kibsoft/amy-mis/pkg/pagination"
	"github.com/kibsoft/amy-mis/pkg/types"
)

// AssignmentHandler handles shift assignment endpoints.
type AssignmentHandler struct {
	assignmentSvc *service.AssignmentService
}

func NewAssignmentHandler(svc *service.AssignmentService) *AssignmentHandler {
	return &AssignmentHandler{assignmentSvc: svc}
}

// Create godoc
// @Summary Create
// @Description Create AssignmentHandler
// @Tags Assignment
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/assignments [post]
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

	// Gap 4: SACCO_ADMIN can only create assignments within their own SACCO
	if claims.SystemRole == types.RoleSaccoAdmin {
		if claims.SaccoID == nil {
			Forbidden(c, "SACCO admin has no SACCO assigned")
			return
		}
		if req.SaccoID != *claims.SaccoID {
			Forbidden(c, "Cannot create assignments for a different SACCO")
			return
		}
	}

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

// Complete godoc
// @Summary Complete
// @Description Complete AssignmentHandler
// @Tags Assignment
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/assignments/{id}/complete [post]
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

// GetByID godoc
// @Summary GetByID
// @Description GetByID AssignmentHandler
// @Tags Assignment
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/assignments/{id} [get]
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

	// Gap 4: SACCO_ADMIN can only view assignments within their SACCO
	claims := middleware.GetClaims(c)
	if claims.SystemRole == types.RoleSaccoAdmin && claims.SaccoID != nil {
		if assignment.SaccoID != *claims.SaccoID {
			Forbidden(c, "Cannot access assignments from a different SACCO")
			return
		}
	}

	SuccessResponse(c, http.StatusOK, dto.AssignmentToResponse(assignment))
}

// List godoc
// @Summary List
// @Description List AssignmentHandler
// @Tags Assignment
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/assignments [get]
func (h *AssignmentHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	filter := repository.AssignmentFilter{}

	// Gap 4: SACCO_ADMIN is automatically scoped to their own SACCO
	claims := middleware.GetClaims(c)
	if claims.SystemRole == types.RoleSaccoAdmin && claims.SaccoID != nil {
		filter.SaccoID = claims.SaccoID
	} else if saccoID := c.Query("sacco_id"); saccoID != "" {
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

// Cancel godoc
// @Summary Cancel an assignment
// @Tags Assignment
// @Param id path string true "Assignment ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/assignments/{id}/cancel [post]
func (h *AssignmentHandler) Cancel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid assignment ID")
		return
	}
	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	result, err := h.assignmentSvc.CancelAssignment(c.Request.Context(), id, req.Reason)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, result)
}

// Reassign godoc
// @Summary Reassign an assignment to a different crew member
// @Tags Assignment
// @Param id path string true "Assignment ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/assignments/{id}/reassign [post]
func (h *AssignmentHandler) Reassign(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid assignment ID")
		return
	}
	var req struct {
		NewCrewMemberID uuid.UUID `json:"new_crew_member_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	result, err := h.assignmentSvc.ReassignAssignment(c.Request.Context(), id, req.NewCrewMemberID)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, result)
}

// --- Wallet Handler ---

// WalletHandler handles wallet and transaction endpoints.
type WalletHandler struct {
	walletSvc  *service.WalletService
	csvMaxRows int
}

func NewWalletHandler(svc *service.WalletService, csvMaxRows int) *WalletHandler {
	return &WalletHandler{walletSvc: svc, csvMaxRows: csvMaxRows}
}

// enforceWalletAccess checks that the requesting user has access to the given crew member's wallet.
// CREW users can only access their own wallet. SACCO_ADMIN can access wallets
// of crew members in their SACCO (not enforced at DB level yet). SYSTEM_ADMIN can access any.
// Returns true if access is denied (response already sent).
func enforceWalletAccess(c *gin.Context, crewMemberID uuid.UUID) bool {
	claims := middleware.GetClaims(c)
	if claims == nil {
		Unauthorized(c, "Authentication required")
		return true
	}

	// SYSTEM_ADMIN can access any wallet
	if claims.SystemRole == types.RoleSystemAdmin {
		return false
	}

	// SACCO_ADMIN can access any wallet (SACCO-level filtering is a future enhancement)
	if claims.SystemRole == types.RoleSaccoAdmin {
		return false
	}

	// CREW users can only access their own wallet
	if claims.SystemRole == types.RoleCrewUser {
		if claims.CrewMemberID == nil || *claims.CrewMemberID != crewMemberID {
			Forbidden(c, "You can only access your own wallet")
			return true
		}
		return false
	}

	// LENDER, INSURER — no wallet access for now
	Forbidden(c, "Insufficient permissions to access wallets")
	return true
}

// GetBalance godoc
// @Summary GetBalance
// @Description GetBalance WalletHandler
// @Tags Wallet
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/wallets/{crew_member_id} [get]
func (h *WalletHandler) GetBalance(c *gin.Context) {
	crewMemberID, err := uuid.Parse(c.Param("crew_member_id"))
	if err != nil {
		BadRequest(c, "Invalid crew member ID")
		return
	}

	// Gap 5: Enforce wallet ownership
	if denied := enforceWalletAccess(c, crewMemberID); denied {
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

// Credit godoc
// @Summary Credit
// @Description Credit WalletHandler
// @Tags Wallet
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/wallets/credit [post]
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

// Debit godoc
// @Summary Debit
// @Description Debit WalletHandler
// @Tags Wallet
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/wallets/debit [post]
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

// ListTransactions godoc
// @Summary ListTransactions
// @Description ListTransactions WalletHandler
// @Tags Wallet
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/wallets/{crew_member_id}/transactions [get]
func (h *WalletHandler) ListTransactions(c *gin.Context) {
	crewMemberID, err := uuid.Parse(c.Param("crew_member_id"))
	if err != nil {
		BadRequest(c, "Invalid crew member ID")
		return
	}

	// Gap 5: Enforce wallet ownership for transaction history too
	if denied := enforceWalletAccess(c, crewMemberID); denied {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	filter := repository.TxFilter{
		Category:        c.Query("category"),
		TransactionType: c.Query("transaction_type"),
		Status:          c.Query("status"),
	}

	if df := c.Query("date_from"); df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			filter.DateFrom = &t
		}
	}
	if dt := c.Query("date_to"); dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			end := t.Add(24*time.Hour - time.Nanosecond) // end of day
			filter.DateTo = &end
		}
	}

	txs, total, err := h.walletSvc.GetTransactions(c.Request.Context(), crewMemberID, filter, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	ListResponse(c, dto.WalletTxListToResponse(txs), buildMeta(page, perPage, total))
}

// --- Payout Handler ---

type PayoutHandler struct {
	payoutSvc *service.PayoutService
}

func NewPayoutHandler(svc *service.PayoutService) *PayoutHandler {
	return &PayoutHandler{payoutSvc: svc}
}

// Payout godoc
// @Summary Payout
// @Description Payout PayoutHandler
// @Tags Payout
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/wallets/{crew_member_id}/payout [post]
func (h *PayoutHandler) Payout(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		BadRequest(c, "Idempotency-Key header is required for financial operations")
		return
	}

	crewMemberID, err := uuid.Parse(c.Param("crew_member_id"))
	if err != nil {
		BadRequest(c, "Invalid crew member ID")
		return
	}

	if denied := enforceWalletAccess(c, crewMemberID); denied {
		return
	}

	var req dto.PayoutWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	result, err := h.payoutSvc.InitiatePayout(c.Request.Context(), service.PayoutInput{
		CrewMemberID:   crewMemberID,
		AmountCents:    req.AmountCents,
		Channel:        payment.PayoutChannel(req.Channel),
		RecipientName:  req.RecipientName,
		RecipientPhone: req.RecipientPhone,
		BankAccount:    req.BankAccount,
		BankCode:       req.BankCode,
		PaybillNumber:  req.PaybillNumber,
		PaybillRef:     req.PaybillRef,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusCreated, result)
}

// ExportCSV godoc
// @Summary Export wallet transactions as CSV
// @Tags Wallet
// @Produce text/csv
// @Param crew_member_id path string true "Crew Member ID"
// @Success 200 {string} string "CSV file"
// @Router /api/v1/wallets/{crew_member_id}/export [get]
func (h *WalletHandler) ExportCSV(c *gin.Context) {
	crewMemberID, err := uuid.Parse(c.Param("crew_member_id"))
	if err != nil {
		BadRequest(c, "Invalid crew member ID")
		return
	}
	if enforceWalletAccess(c, crewMemberID) {
		return
	}

	wallet, err := h.walletSvc.GetBalance(c.Request.Context(), crewMemberID)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	// Use configurable row cap (injected at handler construction, default 10000)
	maxRows := h.csvMaxRows
	if maxRows <= 0 {
		maxRows = 10000
	}
	txns, _, err := h.walletSvc.GetTransactions(c.Request.Context(), wallet.ID, repository.TxFilter{}, 1, maxRows)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=wallet_statement.csv")
	c.Writer.WriteString("Date,Type,Category,Amount (KES),Balance After (KES),Description,Reference\n")
	for _, tx := range txns {
		line := fmt.Sprintf("%s,%s,%s,%.2f,%.2f,%s,%s\n",
			tx.CreatedAt.Format("2006-01-02 15:04:05"),
			tx.TransactionType,
			tx.Category,
			float64(tx.AmountCents)/100.0,
			float64(tx.BalanceAfterCents)/100.0,
			sanitizeCSVCell(tx.Description),
			sanitizeCSVCell(tx.Reference),
		)
		c.Writer.WriteString(line)
	}
}

// sanitizeCSVCell prevents CSV injection by escaping cells that start with
// formula-triggering characters (=, +, -, @, tab, carriage return).
// Also removes commas and newlines from cell content to prevent column/row breakage.
func sanitizeCSVCell(s string) string {
	if len(s) == 0 {
		return s
	}
	// Prefix with a single-quote to prevent formula interpretation in Excel/Sheets
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		s = "'" + s
	}
	// Replace commas and newlines to prevent CSV structure breakage
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ',':
			result = append(result, ' ')
		case '\n', '\r':
			result = append(result, ' ')
		default:
			result = append(result, s[i])
		}
	}
	return string(result)
}

// --- Crew Handler ---

// CrewHandler handles crew member endpoints.
type CrewHandler struct {
	crewSvc *service.CrewService
}

func NewCrewHandler(svc *service.CrewService) *CrewHandler {
	return &CrewHandler{crewSvc: svc}
}

// Create godoc
// @Summary Create
// @Description Create CrewHandler
// @Tags Crew
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/crew [post]
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

// GetByID godoc
// @Summary GetByID
// @Description GetByID CrewHandler
// @Tags Crew
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/crew/{id} [get]
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

// UpdateKYC godoc
// @Summary UpdateKYC
// @Description UpdateKYC CrewHandler
// @Tags Crew
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/crew/{id}/kyc [put]
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
		SerialNumber: req.SerialNumber,
	})
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, dto.CrewToResponse(crew))
}

// List godoc
// @Summary List
// @Description List CrewHandler
// @Tags Crew
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/crew [get]
func (h *CrewHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	filter := repository.CrewFilter{
		Role:      c.Query("role"),
		KYCStatus: c.Query("kyc_status"),
		Search:    c.Query("search"),
	}

	// Gap 4: SACCO_ADMIN is automatically scoped to their own SACCO
	claims := middleware.GetClaims(c)
	if claims.SystemRole == types.RoleSaccoAdmin && claims.SaccoID != nil {
		filter.SaccoID = claims.SaccoID
	} else if saccoID := c.Query("sacco_id"); saccoID != "" {
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

// Deactivate godoc
// @Summary Deactivate
// @Description Deactivate CrewHandler
// @Tags Crew
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/crew/{id} [delete]
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

// VerifyNationalID godoc
// @Summary VerifyNationalID
// @Description VerifyNationalID CrewHandler
// @Tags Crew
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/crew/{id}/verify [post]
func (h *CrewHandler) VerifyNationalID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid crew member ID")
		return
	}

	var req struct {
		SerialNumber string `json:"serial_number" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	crew, err := h.crewSvc.VerifyNationalID(c.Request.Context(), id, req.SerialNumber)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, dto.CrewToResponse(crew))
}

// BulkImport godoc
// @Summary Bulk import crew members
// @Tags Crew
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/crew/bulk-import [post]
func (h *CrewHandler) BulkImport(c *gin.Context) {
	var req struct {
		Members []dto.CreateCrewRequest `json:"members" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	inputs := make([]service.CreateCrewInput, len(req.Members))
	for i, m := range req.Members {
		inputs[i] = service.CreateCrewInput{
			NationalID: m.NationalID,
			FirstName:  m.FirstName,
			LastName:   m.LastName,
			Role:       m.Role,
		}
	}

	result, err := h.crewSvc.BulkImport(c.Request.Context(), inputs)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, result)
}

// SearchByNationalID godoc
// @Summary Search crew member by national ID
// @Tags Crew
// @Produce json
// @Param national_id query string true "National ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/crew/search [get]
func (h *CrewHandler) SearchByNationalID(c *gin.Context) {
	nationalID := c.Query("national_id")
	if nationalID == "" {
		BadRequest(c, "national_id query parameter is required")
		return
	}
	crew, err := h.crewSvc.GetByNationalID(c.Request.Context(), nationalID)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, dto.CrewToResponse(crew))
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

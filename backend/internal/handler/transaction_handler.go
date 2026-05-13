package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/kibsoft/amy-mis/internal/service"
	"github.com/kibsoft/amy-mis/pkg/types"
)

// TransactionHandler handles atomic payout and transfer operations.
type TransactionHandler struct {
	txSvc *service.TransactionService
}

// NewTransactionHandler creates a new TransactionHandler.
func NewTransactionHandler(txSvc *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{txSvc: txSvc}
}

// employeePayoutRequest is the JSON body for the employee payout endpoint.
type employeePayoutRequest struct {
	CrewMemberID   uuid.UUID `json:"crew_member_id" binding:"required"`
	GrossCents     int64     `json:"gross_cents" binding:"required,min=1"`
	NetCents       int64     `json:"net_cents" binding:"required,min=1"`
	IdempotencyKey string    `json:"idempotency_key" binding:"required"`
	Description    string    `json:"description"`
}

// EmployeePayout godoc
// @Summary Atomic employee payout
// @Description Atomically debits org float (gross) and credits employee wallet (net)
// @Tags Transactions
// @Accept json
// @Produce json
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/transactions/employee-payout [post]
func (h *TransactionHandler) EmployeePayout(c *gin.Context) {
	var req employeePayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// Determine org ID from JWT or request context
	claims := middleware.GetClaims(c)
	if claims == nil {
		Forbidden(c, "Authentication required")
		return
	}

	// SACCO_ADMIN: uses their own org. SYSTEM_ADMIN: could pass org_id (future).
	if claims.OrganizationID == nil {
		Forbidden(c, "Organization context required for employee payout")
		return
	}

	result, err := h.txSvc.EmployeePayout(c.Request.Context(), service.EmployeePayoutInput{
		OrganizationID: *claims.OrganizationID,
		CrewMemberID:   req.CrewMemberID,
		GrossCents:     req.GrossCents,
		NetCents:       req.NetCents,
		IdempotencyKey: req.IdempotencyKey,
		Description:    req.Description,
	})
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusCreated, result)
}

// walletTransferRequest is the JSON body for the wallet transfer endpoint.
type walletTransferRequest struct {
	ToCrewMemberID uuid.UUID `json:"to_crew_member_id" binding:"required"`
	AmountCents    int64     `json:"amount_cents" binding:"required,min=1"`
	IdempotencyKey string    `json:"idempotency_key" binding:"required"`
	Description    string    `json:"description"`
}

// WalletTransfer godoc
// @Summary Atomic wallet-to-wallet transfer
// @Description Atomically debits sender wallet and credits recipient wallet
// @Tags Transactions
// @Accept json
// @Produce json
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/transactions/transfer [post]
func (h *TransactionHandler) WalletTransfer(c *gin.Context) {
	var req walletTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// Get sender's crew_member_id from JWT
	claims := middleware.GetClaims(c)
	if claims == nil || claims.CrewMemberID == nil {
		Forbidden(c, "Crew member context required for wallet transfer")
		return
	}

	// Enforce: crew users can only transfer FROM their own wallet
	if claims.SystemRole == types.RoleCrewUser && *claims.CrewMemberID == req.ToCrewMemberID {
		BadRequest(c, "Cannot transfer to yourself")
		return
	}

	result, err := h.txSvc.WalletTransfer(c.Request.Context(), service.WalletTransferInput{
		FromCrewMemberID: *claims.CrewMemberID,
		ToCrewMemberID:   req.ToCrewMemberID,
		AmountCents:      req.AmountCents,
		IdempotencyKey:   req.IdempotencyKey,
		Description:      req.Description,
	})
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusCreated, result)
}

// bulkPayoutItem is a single row in a bulk payout request.
type bulkPayoutItem struct {
	CrewMemberID string `json:"crew_member_id" binding:"required"`
	GrossCents   int64  `json:"gross_cents" binding:"required,min=1"`
	NetCents     int64  `json:"net_cents" binding:"required,min=1"`
	Description  string `json:"description"`
}

// bulkEmployeePayoutRequest is the JSON body for the bulk payout endpoint.
type bulkEmployeePayoutRequest struct {
	// Payouts is the list of individual payout items.
	Payouts []bulkPayoutItem `json:"payouts" binding:"required,min=1"`
	// IdempotencyPrefix is prepended to each item's crew_member_id to form a unique key.
	// Use a unique value per submission (e.g. "bulk-may-2026") to allow safe retries.
	IdempotencyPrefix string `json:"idempotency_prefix" binding:"required"`
}

// bulkPayoutResult holds the per-item outcome.
type bulkPayoutResult struct {
	CrewMemberID string `json:"crew_member_id"`
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"`
}

// BulkEmployeePayout godoc
// @Summary Bulk atomic employee payout
// @Description Processes multiple employee payouts sequentially, each atomically debiting org float (gross) and crediting employee wallet (net). Returns per-item success/failure.
// @Tags Transactions
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/transactions/bulk-employee-payout [post]
func (h *TransactionHandler) BulkEmployeePayout(c *gin.Context) {
	var req bulkEmployeePayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		Forbidden(c, "Authentication required")
		return
	}
	if claims.OrganizationID == nil {
		Forbidden(c, "Organization context required for bulk employee payout")
		return
	}

	ctx := c.Request.Context()
	orgID := *claims.OrganizationID

	var succeeded []bulkPayoutResult
	var failed []bulkPayoutResult

	for _, item := range req.Payouts {
		crewID, parseErr := uuid.Parse(item.CrewMemberID)
		if parseErr != nil {
			failed = append(failed, bulkPayoutResult{
				CrewMemberID: item.CrewMemberID,
				Success:      false,
				Error:        "invalid crew_member_id: " + parseErr.Error(),
			})
			continue
		}

		idempotencyKey := req.IdempotencyPrefix + ":" + item.CrewMemberID
		_, err := h.txSvc.EmployeePayout(ctx, service.EmployeePayoutInput{
			OrganizationID: orgID,
			CrewMemberID:   crewID,
			GrossCents:     item.GrossCents,
			NetCents:       item.NetCents,
			IdempotencyKey: idempotencyKey,
			Description:    item.Description,
		})
		if err != nil {
			failed = append(failed, bulkPayoutResult{
				CrewMemberID: item.CrewMemberID,
				Success:      false,
				Error:        err.Error(),
			})
		} else {
			succeeded = append(succeeded, bulkPayoutResult{
				CrewMemberID: item.CrewMemberID,
				Success:      true,
			})
		}
	}

	SuccessResponse(c, http.StatusOK, gin.H{
		"total":     len(req.Payouts),
		"succeeded": len(succeeded),
		"failed":    len(failed),
		"results": gin.H{
			"succeeded": succeeded,
			"failed":    failed,
		},
	})
}

package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/external/payment"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/internal/service"
)

// --- SACCO Handler ---

type OrganizationHandler struct {
	saccoSvc   *service.OrganizationService
	paymentMgr *payment.Manager // nil when no payment providers configured
}

func NewOrganizationHandler(svc *service.OrganizationService, paymentMgr ...*payment.Manager) *OrganizationHandler {
	h := &OrganizationHandler{saccoSvc: svc}
	if len(paymentMgr) > 0 {
		h.paymentMgr = paymentMgr[0]
	}
	return h
}

func (h *OrganizationHandler) Create(c *gin.Context) {
	var req service.CreateSACCOInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	sacco, err := h.saccoSvc.CreateSACCO(c.Request.Context(), req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, sacco)
}

func (h *OrganizationHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid SACCO ID")
		return
	}
	sacco, err := h.saccoSvc.GetSACCO(c.Request.Context(), id)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, sacco)
}

func (h *OrganizationHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid SACCO ID")
		return
	}
	var req service.UpdateSACCOInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	sacco, err := h.saccoSvc.UpdateSACCO(c.Request.Context(), id, req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, sacco)
}

func (h *OrganizationHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid SACCO ID")
		return
	}
	if err := h.saccoSvc.DeleteSACCO(c.Request.Context(), id); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, gin.H{"message": "SACCO deleted"})
}

func (h *OrganizationHandler) List(c *gin.Context) {
	// SACCO_ADMIN: only return their own organization
	claims := middleware.GetClaims(c)
	if claims != nil && claims.SystemRole == "SACCO_ADMIN" && claims.OrganizationID != nil {
		sacco, err := h.saccoSvc.GetSACCO(c.Request.Context(), *claims.OrganizationID)
		if err != nil {
			MapServiceError(c, err)
			return
		}
		ListResponse(c, []models.SACCO{*sacco}, buildMeta(1, 1, 1))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	search := c.Query("search")

	saccos, total, err := h.saccoSvc.ListSACCOs(c.Request.Context(), page, perPage, search)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	ListResponse(c, saccos, buildMeta(page, perPage, total))
}

func (h *OrganizationHandler) AddMember(c *gin.Context) {
	var req service.AddMemberInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid SACCO ID")
		return
	}
	req.OrganizationID = orgID

	m, err := h.saccoSvc.AddMember(c.Request.Context(), req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, m)
}

func (h *OrganizationHandler) UpdateMember(c *gin.Context) {
	membershipID, err := uuid.Parse(c.Param("membership_id"))
	if err != nil {
		BadRequest(c, "Invalid membership ID")
		return
	}
	var req service.UpdateMemberInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	m, err := h.saccoSvc.UpdateMember(c.Request.Context(), membershipID, req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, m)
}

func (h *OrganizationHandler) RemoveMember(c *gin.Context) {
	membershipID, err := uuid.Parse(c.Param("membership_id"))
	if err != nil {
		BadRequest(c, "Invalid membership ID")
		return
	}
	if err := h.saccoSvc.RemoveMember(c.Request.Context(), membershipID); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, gin.H{"message": "Member removed"})
}

func (h *OrganizationHandler) ListMembers(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid SACCO ID")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	members, total, err := h.saccoSvc.ListMembers(c.Request.Context(), orgID, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	ListResponse(c, members, buildMeta(page, perPage, total))
}

func (h *OrganizationHandler) GetFloat(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid SACCO ID")
		return
	}
	sf, err := h.saccoSvc.GetFloat(c.Request.Context(), orgID)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, sf)
}

func (h *OrganizationHandler) CreditFloat(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid SACCO ID")
		return
	}
	var req service.FloatOperationInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	req.OrganizationID = orgID
	tx, err := h.saccoSvc.CreditFloat(c.Request.Context(), req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, tx)
}

// TopUpFloat handles organization float top-up.
// For mobile_money: creates a PENDING float transaction, triggers STK push,
// and returns immediately. The float balance is credited only when the payment
// callback confirms success (via webhook handler).
// For bank/card: credits the float immediately (manual entry).
// POST /organizations/:id/float/topup
func (h *OrganizationHandler) TopUpFloat(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid Organization ID")
		return
	}

	var req struct {
		AmountCents    int64  `json:"amount_cents" binding:"required,min=1"`
		IdempotencyKey string `json:"idempotency_key" binding:"required"`
		Method         string `json:"method" binding:"required"`   // "mobile_money", "bank", "card"
		Provider       string `json:"provider"`                    // "mpesa", "airtel", "kcb", "equity", etc.
		PhoneNumber    string `json:"phone_number"`                // Required for mobile_money
		Reference      string `json:"reference"`                   // Reference/description
		BankRef        string `json:"bank_ref"`                    // Bank transfer reference
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	switch req.Method {
	case "mobile_money":
		// --- Async STK push flow ---
		if req.PhoneNumber == "" {
			BadRequest(c, "phone_number is required for mobile money top-up")
			return
		}
		providerLabel := req.Provider
		if providerLabel == "" {
			providerLabel = "mpesa"
		}

		// Build reference
		ref := "STK:" + providerLabel + " | phone:" + req.PhoneNumber
		if req.Reference != "" {
			ref += " | " + req.Reference
		}

		// 1. Create PENDING float transaction (no balance change yet)
		pendingTx, err := h.saccoSvc.CreatePendingTopUp(c.Request.Context(), service.FloatOperationInput{
			OrganizationID: orgID,
			AmountCents:    req.AmountCents,
			IdempotencyKey: req.IdempotencyKey,
			Reference:      ref,
		})
		if err != nil {
			MapServiceError(c, err)
			return
		}

		// 2. Trigger STK push via JamboPay (non-blocking for the response)
		var stkStatus string
		if h.paymentMgr != nil {
			collResult, collErr := h.paymentMgr.InitiateCollection(c.Request.Context(), payment.CollectionRequest{
				AmountCents: req.AmountCents,
				OrderID:     req.IdempotencyKey,
				Provider:    req.Provider,
				PhoneNumber: req.PhoneNumber,
				Description: "Organization float top-up",
			})
			if collErr != nil {
				// STK push failed to send — mark the pending tx as failed
				_ = h.saccoSvc.FailPendingTopUp(c.Request.Context(), pendingTx.ID, collErr.Error())
				ErrorResponse(c, http.StatusBadGateway, "STK_PUSH_FAILED",
					"Failed to initiate M-Pesa STK push: "+collErr.Error())
				return
			}
			stkStatus = "STK push sent. Check your phone (" + req.PhoneNumber + ") to complete payment."
			// Append provider reference to the pending tx reference
			_ = h.saccoSvc.UpdatePendingRef(c.Request.Context(), pendingTx.ID, " | jp_ref:"+collResult.Reference)
		} else {
			stkStatus = "Payment provider not configured"
			_ = h.saccoSvc.FailPendingTopUp(c.Request.Context(), pendingTx.ID, "no payment provider")
			ErrorResponse(c, http.StatusServiceUnavailable, "PROVIDER_UNAVAILABLE",
				"Mobile payment provider is not configured")
			return
		}

		// Return PENDING status — balance NOT yet credited
		SuccessResponse(c, http.StatusAccepted, gin.H{
			"status":         "PENDING",
			"message":        stkStatus,
			"transaction_id": pendingTx.ID,
			"amount_cents":   req.AmountCents,
			"phone_number":   req.PhoneNumber,
		})

	case "bank":
		// --- Immediate credit for bank transfers (manual entry) ---
		bankRef := req.BankRef
		if bankRef == "" {
			bankRef = "pending"
		}
		providerLabel := req.Provider
		if providerLabel == "" {
			providerLabel = "bank"
		}
		ref := "BANK:" + providerLabel + " | txn_ref:" + bankRef
		if req.Reference != "" {
			ref += " | " + req.Reference
		}

		tx, err := h.saccoSvc.CreditFloat(c.Request.Context(), service.FloatOperationInput{
			OrganizationID: orgID,
			AmountCents:    req.AmountCents,
			IdempotencyKey: req.IdempotencyKey,
			Reference:      ref,
		})
		if err != nil {
			MapServiceError(c, err)
			return
		}
		SuccessResponse(c, http.StatusCreated, tx)

	case "card":
		// --- Immediate credit for card payments (manual entry) ---
		ref := "CARD:" + req.Provider
		if req.Reference != "" {
			ref += " | " + req.Reference
		}

		tx, err := h.saccoSvc.CreditFloat(c.Request.Context(), service.FloatOperationInput{
			OrganizationID: orgID,
			AmountCents:    req.AmountCents,
			IdempotencyKey: req.IdempotencyKey,
			Reference:      ref,
		})
		if err != nil {
			MapServiceError(c, err)
			return
		}
		SuccessResponse(c, http.StatusCreated, tx)

	default:
		BadRequest(c, "invalid method: must be mobile_money, bank, or card")
	}
}

func (h *OrganizationHandler) DebitFloat(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid SACCO ID")
		return
	}
	var req service.FloatOperationInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	req.OrganizationID = orgID
	tx, err := h.saccoSvc.DebitFloat(c.Request.Context(), req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, tx)
}

func (h *OrganizationHandler) ListFloatTransactions(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid SACCO ID")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	
	filter := repository.OrganizationFloatFilter{
		TransactionType: c.Query("type"),
	}

	txs, total, err := h.saccoSvc.ListFloatTransactions(c.Request.Context(), orgID, filter, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	ListResponse(c, txs, buildMeta(page, perPage, total))
}

// --- Vehicle Handler ---

type VehicleHandler struct {
	vehicleSvc *service.VehicleService
}

func NewVehicleHandler(svc *service.VehicleService) *VehicleHandler {
	return &VehicleHandler{vehicleSvc: svc}
}

// Create godoc
// @Summary Create
// @Description Create VehicleHandler
// @Tags Vehicle
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/vehicles [post]
func (h *VehicleHandler) Create(c *gin.Context) {
	var req service.CreateVehicleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	// Auto-populate org ID from JWT claims if not provided in body
	if req.OrganizationID == uuid.Nil {
		claims := middleware.GetClaims(c)
		if claims != nil && claims.OrganizationID != nil {
			req.OrganizationID = *claims.OrganizationID
		}
	}
	if req.OrganizationID == uuid.Nil {
		BadRequest(c, "organization_id (sacco_id) is required")
		return
	}
	vehicle, err := h.vehicleSvc.CreateVehicle(c.Request.Context(), req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, vehicle)
}

// GetByID godoc
// @Summary GetByID
// @Description GetByID VehicleHandler
// @Tags Vehicle
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/vehicles/{id} [get]
func (h *VehicleHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid vehicle ID")
		return
	}
	vehicle, err := h.vehicleSvc.GetVehicle(c.Request.Context(), id)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, vehicle)
}

// Update godoc
// @Summary Update
// @Description Update VehicleHandler
// @Tags Vehicle
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/vehicles/{id} [put]
func (h *VehicleHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid vehicle ID")
		return
	}
	var req service.UpdateVehicleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	vehicle, err := h.vehicleSvc.UpdateVehicle(c.Request.Context(), id, req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, vehicle)
}

// Delete godoc
// @Summary Delete
// @Description Delete VehicleHandler
// @Tags Vehicle
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/vehicles/{id} [delete]
func (h *VehicleHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid vehicle ID")
		return
	}
	if err := h.vehicleSvc.DeleteVehicle(c.Request.Context(), id); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, gin.H{"message": "Vehicle deleted"})
}

// List godoc
// @Summary List
// @Description List VehicleHandler
// @Tags Vehicle
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/vehicles [get]
func (h *VehicleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	var orgID *uuid.UUID
	// SACCO_ADMIN: auto-scope to their own organization
	claims := middleware.GetClaims(c)
	if claims != nil && claims.SystemRole == "SACCO_ADMIN" && claims.OrganizationID != nil {
		orgID = claims.OrganizationID
	} else if s := c.Query("sacco_id"); s != "" {
		id, _ := uuid.Parse(s)
		orgID = &id
	}
	vehicles, total, err := h.vehicleSvc.ListVehicles(c.Request.Context(), orgID, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	ListResponse(c, vehicles, buildMeta(page, perPage, total))
}

// --- Route Handler ---

type RouteHandler struct {
	routeSvc *service.RouteService
}

func NewRouteHandler(svc *service.RouteService) *RouteHandler {
	return &RouteHandler{routeSvc: svc}
}

// Create godoc
// @Summary Create
// @Description Create RouteHandler
// @Tags Route
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/routes [post]
func (h *RouteHandler) Create(c *gin.Context) {
	var req service.CreateRouteInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	route, err := h.routeSvc.CreateRoute(c.Request.Context(), req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, route)
}

// GetByID godoc
// @Summary GetByID
// @Description GetByID RouteHandler
// @Tags Route
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/routes/{id} [get]
func (h *RouteHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid route ID")
		return
	}
	route, err := h.routeSvc.GetRoute(c.Request.Context(), id)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, route)
}

// Update godoc
// @Summary Update
// @Description Update RouteHandler
// @Tags Route
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/routes/{id} [put]
func (h *RouteHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid route ID")
		return
	}
	var req service.UpdateRouteInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	route, err := h.routeSvc.UpdateRoute(c.Request.Context(), id, req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, route)
}

// Delete godoc
// @Summary Delete
// @Description Delete RouteHandler
// @Tags Route
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/routes/{id} [delete]
func (h *RouteHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid route ID")
		return
	}
	if err := h.routeSvc.DeleteRoute(c.Request.Context(), id); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, gin.H{"message": "Route deleted"})
}

// List godoc
// @Summary List
// @Description List RouteHandler
// @Tags Route
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/routes [get]
func (h *RouteHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	search := c.Query("search")
	routes, total, err := h.routeSvc.ListRoutes(c.Request.Context(), page, perPage, search)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	ListResponse(c, routes, buildMeta(page, perPage, total))
}

// --- Payroll Handler ---

type PayrollHandler struct {
	payrollSvc *service.PayrollService
}

func NewPayrollHandler(svc *service.PayrollService) *PayrollHandler {
	return &PayrollHandler{payrollSvc: svc}
}

// Create godoc
// @Summary Create
// @Description Create PayrollHandler
// @Tags Payroll
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/payroll [post]
func (h *PayrollHandler) Create(c *gin.Context) {
	var req service.CreatePayrollRunInput
	if err := c.ShouldBindJSON(&req); err != nil {
		// Auto-inject organization_id from JWT for SACCO_ADMIN users
		claims := GetClaimsFromContext(c)
		if claims != nil && claims.OrganizationID != nil && req.OrganizationID == uuid.Nil {
			req.OrganizationID = *claims.OrganizationID
			// Re-validate remaining fields
			if req.PeriodStart == "" || req.PeriodEnd == "" {
				BadRequest(c, err.Error())
				return
			}
		} else {
			BadRequest(c, err.Error())
			return
		}
	}
	// Fallback: inject from JWT if still empty
	if req.OrganizationID == uuid.Nil {
		claims := GetClaimsFromContext(c)
		if claims != nil && claims.OrganizationID != nil {
			req.OrganizationID = *claims.OrganizationID
		}
	}
	run, err := h.payrollSvc.CreatePayrollRun(c.Request.Context(), req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, run)
}

// GetByID godoc
// @Summary GetByID
// @Description GetByID PayrollHandler
// @Tags Payroll
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/payroll/{id} [get]
func (h *PayrollHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid payroll run ID")
		return
	}
	run, err := h.payrollSvc.GetPayrollRun(c.Request.Context(), id)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, run)
}

// List godoc
// @Summary List
// @Description List PayrollHandler
// @Tags Payroll
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/payroll [get]
func (h *PayrollHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	var orgID *uuid.UUID
	// SACCO_ADMIN: auto-scope to their own organization
	claims := middleware.GetClaims(c)
	if claims != nil && claims.SystemRole == "SACCO_ADMIN" && claims.OrganizationID != nil {
		orgID = claims.OrganizationID
	} else if s := c.Query("sacco_id"); s != "" {
		id, _ := uuid.Parse(s)
		orgID = &id
	}
	runs, total, err := h.payrollSvc.ListPayrollRuns(c.Request.Context(), orgID, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	ListResponse(c, runs, buildMeta(page, perPage, total))
}

// GetEntries godoc
// @Summary GetEntries
// @Description GetEntries PayrollHandler
// @Tags Payroll
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/payroll/{id}/entries [get]
func (h *PayrollHandler) GetEntries(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid payroll run ID")
		return
	}
	entries, err := h.payrollSvc.GetPayrollEntries(c.Request.Context(), id)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, entries)
}

// Process godoc
// @Summary Process
// @Description Process PayrollHandler
// @Tags Payroll
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/payroll/{id}/process [post]
func (h *PayrollHandler) Process(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid payroll run ID")
		return
	}
	run, err := h.payrollSvc.ProcessPayrollRun(c.Request.Context(), id)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, run)
}

// Approve godoc
// @Summary Approve
// @Description Approve PayrollHandler
// @Tags Payroll
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/payroll/{id}/approve [post]
func (h *PayrollHandler) Approve(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid payroll run ID")
		return
	}

	// Extract approver identity from JWT claims — never trust client-provided approver IDs
	claims := middleware.GetClaims(c)
	if claims == nil {
		Unauthorized(c, "Authentication required")
		return
	}

	run, err := h.payrollSvc.ApprovePayrollRun(c.Request.Context(), id, claims.UserID)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, run)
}

// Submit godoc
// @Summary Submit
// @Description Submit PayrollHandler — submits an approved payroll run to the external payroll provider (PerPay)
// @Tags Payroll
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/payroll/{id}/submit [post]
func (h *PayrollHandler) Submit(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid payroll run ID")
		return
	}
	run, err := h.payrollSvc.SubmitPayrollRun(c.Request.Context(), id)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, run)
}

// ListPeriods godoc
// @Summary ListPeriods
// @Description List pay periods for an organization
// @Tags Payroll
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/payroll/periods [get]
func (h *PayrollHandler) ListPeriods(c *gin.Context) {
	orgIDStr := c.Query("organization_id")
	if orgIDStr == "" {
		claims := GetClaimsFromContext(c)
		if claims != nil && claims.OrganizationID != nil {
			orgIDStr = claims.OrganizationID.String()
		}
	}
	if orgIDStr == "" {
		BadRequest(c, "organization_id is required")
		return
	}
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		BadRequest(c, "Invalid organization_id")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "100"))
	periods, total, err := h.payrollSvc.ListPayPeriodsByOrg(c.Request.Context(), orgID, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	ListResponse(c, periods, buildMeta(page, perPage, total))
}
// --- Notification Handler ---

type NotificationHandler struct {
	notifSvc *service.NotificationService
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifSvc: svc}
}

func (h *NotificationHandler) List(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		Unauthorized(c, "Authentication required")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	filter := repository.NotificationFilter{
		Channel: c.Query("channel"),
		Status:  c.Query("status"),
	}

	notifs, total, err := h.notifSvc.ListNotifications(c.Request.Context(), claims.UserID,
		filter, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	ListResponse(c, notifs, buildMeta(page, perPage, total))
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid notification ID")
		return
	}
	if err := h.notifSvc.MarkRead(c.Request.Context(), id); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, gin.H{"message": "Notification marked as read"})
}

func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		Unauthorized(c, "Authentication required")
		return
	}
	prefs, err := h.notifSvc.GetPreferences(c.Request.Context(), claims.UserID)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, prefs)
}

func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		Unauthorized(c, "Authentication required")
		return
	}
	var p models.NotificationPreference
	if err := c.ShouldBindJSON(&p); err != nil {
		BadRequest(c, err.Error())
		return
	}
	p.UserID = claims.UserID
	if err := h.notifSvc.UpdatePreferences(c.Request.Context(), &p); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, p)
}

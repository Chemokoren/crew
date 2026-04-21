package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/internal/service"
)

// --- SACCO Handler ---

type SACCOHandler struct {
	saccoSvc *service.SACCOService
}

func NewSACCOHandler(svc *service.SACCOService) *SACCOHandler {
	return &SACCOHandler{saccoSvc: svc}
}

func (h *SACCOHandler) Create(c *gin.Context) {
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

func (h *SACCOHandler) GetByID(c *gin.Context) {
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

func (h *SACCOHandler) Update(c *gin.Context) {
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

func (h *SACCOHandler) Delete(c *gin.Context) {
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

func (h *SACCOHandler) List(c *gin.Context) {
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

func (h *SACCOHandler) AddMember(c *gin.Context) {
	var req service.AddMemberInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	saccoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid SACCO ID")
		return
	}
	req.SaccoID = saccoID

	m, err := h.saccoSvc.AddMember(c.Request.Context(), req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, m)
}

func (h *SACCOHandler) RemoveMember(c *gin.Context) {
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

func (h *SACCOHandler) ListMembers(c *gin.Context) {
	saccoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid SACCO ID")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	members, total, err := h.saccoSvc.ListMembers(c.Request.Context(), saccoID, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	ListResponse(c, members, buildMeta(page, perPage, total))
}

func (h *SACCOHandler) GetFloat(c *gin.Context) {
	saccoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid SACCO ID")
		return
	}
	sf, err := h.saccoSvc.GetFloat(c.Request.Context(), saccoID)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, sf)
}

func (h *SACCOHandler) CreditFloat(c *gin.Context) {
	saccoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid SACCO ID")
		return
	}
	var req service.FloatOperationInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	req.SaccoID = saccoID
	tx, err := h.saccoSvc.CreditFloat(c.Request.Context(), req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, tx)
}

func (h *SACCOHandler) DebitFloat(c *gin.Context) {
	saccoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid SACCO ID")
		return
	}
	var req service.FloatOperationInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	req.SaccoID = saccoID
	tx, err := h.saccoSvc.DebitFloat(c.Request.Context(), req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, tx)
}

// --- Vehicle Handler ---

type VehicleHandler struct {
	vehicleSvc *service.VehicleService
}

func NewVehicleHandler(svc *service.VehicleService) *VehicleHandler {
	return &VehicleHandler{vehicleSvc: svc}
}

func (h *VehicleHandler) Create(c *gin.Context) {
	var req service.CreateVehicleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	vehicle, err := h.vehicleSvc.CreateVehicle(c.Request.Context(), req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, vehicle)
}

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

func (h *VehicleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	var saccoID *uuid.UUID
	if s := c.Query("sacco_id"); s != "" {
		id, _ := uuid.Parse(s)
		saccoID = &id
	}
	vehicles, total, err := h.vehicleSvc.ListVehicles(c.Request.Context(), saccoID, page, perPage)
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

func (h *PayrollHandler) Create(c *gin.Context) {
	var req service.CreatePayrollRunInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	run, err := h.payrollSvc.CreatePayrollRun(c.Request.Context(), req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, run)
}

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

func (h *PayrollHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	var saccoID *uuid.UUID
	if s := c.Query("sacco_id"); s != "" {
		id, _ := uuid.Parse(s)
		saccoID = &id
	}
	runs, total, err := h.payrollSvc.ListPayrollRuns(c.Request.Context(), saccoID, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	ListResponse(c, runs, buildMeta(page, perPage, total))
}

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

func (h *PayrollHandler) Approve(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid payroll run ID")
		return
	}
	// Use a dummy approver for now — in production, extract from JWT claims
	var req struct {
		ApproverID uuid.UUID `json:"approver_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	run, err := h.payrollSvc.ApprovePayrollRun(c.Request.Context(), id, req.ApproverID)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, run)
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

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/service"
)

// TenantHandler manages tenant configuration REST endpoints.
type TenantHandler struct {
	tenantSvc *service.TenantService
}

// NewTenantHandler creates a new TenantHandler.
func NewTenantHandler(svc *service.TenantService) *TenantHandler {
	return &TenantHandler{tenantSvc: svc}
}

// --- Tenant Config ---

// GetConfig godoc
// @Summary Get tenant configuration
// @Description Returns the full tenant configuration including industry type, config, and display name
// @Tags Tenant
// @Produce json
// @Param id path string true "SACCO/Organization ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tenants/{id}/config [get]
func (h *TenantHandler) GetConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid tenant ID")
		return
	}
	sacco, err := h.tenantSvc.GetTenantConfig(c.Request.Context(), id)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, sacco)
}

// UpdateConfig godoc
// @Summary Update tenant configuration
// @Description Updates industry type, display name, and/or tenant config
// @Tags Tenant
// @Accept json
// @Produce json
// @Param id path string true "SACCO/Organization ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tenants/{id}/config [put]
func (h *TenantHandler) UpdateConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid tenant ID")
		return
	}
	var req service.UpdateTenantConfigInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	sacco, err := h.tenantSvc.UpdateTenantConfig(c.Request.Context(), id, req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, sacco)
}

// --- Job Types ---

// ListJobTypes godoc
// @Summary List job types for a tenant
// @Description Returns all active job types configured for the organization
// @Tags Tenant
// @Produce json
// @Param id path string true "SACCO/Organization ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tenants/{id}/job-types [get]
func (h *TenantHandler) ListJobTypes(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid tenant ID")
		return
	}
	jobTypes, err := h.tenantSvc.ListJobTypes(c.Request.Context(), orgID)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, jobTypes)
}

// CreateJobType godoc
// @Summary Create a job type
// @Description Creates a new configurable job type for the organization
// @Tags Tenant
// @Accept json
// @Produce json
// @Param id path string true "SACCO/Organization ID"
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/tenants/{id}/job-types [post]
func (h *TenantHandler) CreateJobType(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid tenant ID")
		return
	}
	var req service.CreateJobTypeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	req.OrganizationID = orgID
	jt, err := h.tenantSvc.CreateJobType(c.Request.Context(), req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, jt)
}

// UpdateJobType godoc
// @Summary Update a job type
// @Description Updates an existing job type
// @Tags Tenant
// @Accept json
// @Produce json
// @Param id path string true "SACCO/Organization ID"
// @Param job_type_id path string true "Job Type ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tenants/{id}/job-types/{job_type_id} [put]
func (h *TenantHandler) UpdateJobType(c *gin.Context) {
	jtID, err := uuid.Parse(c.Param("job_type_id"))
	if err != nil {
		BadRequest(c, "Invalid job type ID")
		return
	}
	var req service.UpdateJobTypeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	jt, err := h.tenantSvc.UpdateJobType(c.Request.Context(), jtID, req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, jt)
}

// DeleteJobType godoc
// @Summary Delete a job type
// @Description Deletes a job type from the organization
// @Tags Tenant
// @Produce json
// @Param id path string true "SACCO/Organization ID"
// @Param job_type_id path string true "Job Type ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tenants/{id}/job-types/{job_type_id} [delete]
func (h *TenantHandler) DeleteJobType(c *gin.Context) {
	jtID, err := uuid.Parse(c.Param("job_type_id"))
	if err != nil {
		BadRequest(c, "Invalid job type ID")
		return
	}
	if err := h.tenantSvc.DeleteJobType(c.Request.Context(), jtID); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, gin.H{"message": "Job type deleted"})
}

// --- Pay Schedules ---

// ListPaySchedules godoc
// @Summary List pay schedules for a tenant
// @Description Returns all active pay schedules configured for the organization
// @Tags Tenant
// @Produce json
// @Param id path string true "SACCO/Organization ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tenants/{id}/pay-schedules [get]
func (h *TenantHandler) ListPaySchedules(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid tenant ID")
		return
	}
	schedules, err := h.tenantSvc.ListPaySchedules(c.Request.Context(), orgID)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, schedules)
}

// CreatePaySchedule godoc
// @Summary Create a pay schedule
// @Description Creates a new pay schedule for the organization
// @Tags Tenant
// @Accept json
// @Produce json
// @Param id path string true "SACCO/Organization ID"
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/tenants/{id}/pay-schedules [post]
func (h *TenantHandler) CreatePaySchedule(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid tenant ID")
		return
	}
	var req service.CreatePayScheduleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	req.OrganizationID = orgID
	ps, err := h.tenantSvc.CreatePaySchedule(c.Request.Context(), req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, ps)
}

// UpdatePaySchedule godoc
// @Summary Update a pay schedule
// @Description Updates an existing pay schedule
// @Tags Tenant
// @Accept json
// @Produce json
// @Param id path string true "SACCO/Organization ID"
// @Param schedule_id path string true "Pay Schedule ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tenants/{id}/pay-schedules/{schedule_id} [put]
func (h *TenantHandler) UpdatePaySchedule(c *gin.Context) {
	scheduleID, err := uuid.Parse(c.Param("schedule_id"))
	if err != nil {
		BadRequest(c, "Invalid pay schedule ID")
		return
	}
	var req service.UpdatePayScheduleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	ps, err := h.tenantSvc.UpdatePaySchedule(c.Request.Context(), scheduleID, req)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, ps)
}

// DeletePaySchedule godoc
// @Summary Delete a pay schedule
// @Description Deletes a pay schedule from the organization
// @Tags Tenant
// @Produce json
// @Param id path string true "SACCO/Organization ID"
// @Param schedule_id path string true "Pay Schedule ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tenants/{id}/pay-schedules/{schedule_id} [delete]
func (h *TenantHandler) DeletePaySchedule(c *gin.Context) {
	scheduleID, err := uuid.Parse(c.Param("schedule_id"))
	if err != nil {
		BadRequest(c, "Invalid pay schedule ID")
		return
	}
	if err := h.tenantSvc.DeletePaySchedule(c.Request.Context(), scheduleID); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, gin.H{"message": "Pay schedule deleted"})
}

// --- Industry Bootstrap (AD-13) ---

// BootstrapIndustry godoc
// @Summary Bootstrap industry configuration
// @Description Seeds default job types, pay schedules, and config from an industry template
// @Tags Tenant
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tenants/{id}/bootstrap [post]
func (h *TenantHandler) BootstrapIndustry(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid organization ID")
		return
	}
	var req struct {
		IndustryType string `json:"industry_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	result, err := h.tenantSvc.BootstrapIndustry(c.Request.Context(), orgID, models.IndustryType(req.IndustryType))
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, result)
}

// GetIndustryTemplate godoc
// @Summary Get industry template
// @Description Returns the pre-configured template for an industry type
// @Tags Tenant
// @Produce json
// @Param industry query string true "Industry type (TRANSPORT, CONSTRUCTION, etc.)"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/industry-templates [get]
func (h *TenantHandler) GetIndustryTemplate(c *gin.Context) {
	industry := c.Query("industry")
	if industry == "" {
		// Return all templates
		all := make(map[string]models.IndustryTemplate)
		for _, it := range []models.IndustryType{
			models.IndustryTransport, models.IndustryConstruction, models.IndustryHealth,
			models.IndustryLogistics, models.IndustryAgriculture, models.IndustryHospitality,
			models.IndustryGeneral,
		} {
			all[string(it)] = models.GetIndustryTemplate(it)
		}
		SuccessResponse(c, http.StatusOK, all)
		return
	}
	template := models.GetIndustryTemplate(models.IndustryType(industry))
	SuccessResponse(c, http.StatusOK, template)
}

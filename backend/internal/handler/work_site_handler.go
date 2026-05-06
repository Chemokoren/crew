package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/pkg/errs"
	"github.com/kibsoft/amy-mis/pkg/pagination"
	"github.com/kibsoft/amy-mis/pkg/types"
)

// WorkSiteHandler handles CRUD for named work sites / project locations.
type WorkSiteHandler struct {
	repo repository.WorkSiteRepository
}

func NewWorkSiteHandler(repo repository.WorkSiteRepository) *WorkSiteHandler {
	return &WorkSiteHandler{repo: repo}
}

// List godoc
// @Summary List work sites
// @Tags WorkSites
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/work-sites [get]
func (h *WorkSiteHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "50"))
	search := c.Query("search")

	claims := middleware.GetClaims(c)

	var orgID *uuid.UUID
	// SACCO_ADMIN is always scoped to their own org
	if claims.SystemRole == types.RoleSaccoAdmin && claims.OrganizationID != nil {
		orgID = claims.OrganizationID
	} else if q := c.Query("organization_id"); q != "" {
		id, err := uuid.Parse(q)
		if err == nil {
			orgID = &id
		}
	}

	sites, total, err := h.repo.List(c.Request.Context(), orgID, page, perPage, search)
	if err != nil {
		InternalError(c, "Failed to list work sites")
		return
	}

	responses := make([]workSiteResponse, len(sites))
	for i, s := range sites {
		responses[i] = toWorkSiteResponse(&s)
	}

	totalInt := int(total)
	totalPages := totalInt / perPage
	if totalInt%perPage != 0 {
		totalPages++
	}
	ListResponse(c, responses, pagination.Meta{Page: page, PerPage: perPage, Total: totalInt, TotalPages: totalPages})
}

// GetByID godoc
// @Summary Get a work site by ID
// @Tags WorkSites
// @Produce json
// @Param id path string true "Work Site ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/work-sites/{id} [get]
func (h *WorkSiteHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid work site ID")
		return
	}
	site, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			NotFound(c, "Work site not found")
		} else {
			InternalError(c, "Failed to get work site")
		}
		return
	}
	SuccessResponse(c, http.StatusOK, toWorkSiteResponse(site))
}

// Create godoc
// @Summary Create a work site
// @Tags WorkSites
// @Accept json
// @Produce json
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/work-sites [post]
func (h *WorkSiteHandler) Create(c *gin.Context) {
	var req struct {
		Name           string    `json:"name" binding:"required"`
		ProjectRef     string    `json:"project_ref"`
		Address        string    `json:"address"`
		Description    string    `json:"description"`
		OrganizationID uuid.UUID `json:"organization_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	claims := middleware.GetClaims(c)

	// Auto-fill org from JWT for SACCO_ADMIN
	orgID := req.OrganizationID
	if claims.SystemRole == types.RoleSaccoAdmin && claims.OrganizationID != nil {
		orgID = *claims.OrganizationID
	}
	if orgID == uuid.Nil {
		BadRequest(c, "organization_id is required")
		return
	}

	site := &models.WorkSite{
		OrganizationID: orgID,
		Name:           req.Name,
		ProjectRef:     req.ProjectRef,
		Address:        req.Address,
		Description:    req.Description,
		IsActive:       true,
		CreatedByID:    claims.UserID,
	}

	if err := h.repo.Create(c.Request.Context(), site); err != nil {
		InternalError(c, "Failed to create work site")
		return
	}
	SuccessResponse(c, http.StatusCreated, toWorkSiteResponse(site))
}

// Update godoc
// @Summary Update a work site
// @Tags WorkSites
// @Accept json
// @Produce json
// @Param id path string true "Work Site ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/work-sites/{id} [put]
func (h *WorkSiteHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid work site ID")
		return
	}

	site, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			NotFound(c, "Work site not found")
		} else {
			InternalError(c, "Failed to get work site")
		}
		return
	}

	// SACCO_ADMIN can only update their own org's sites
	claims := middleware.GetClaims(c)
	if claims.SystemRole == types.RoleSaccoAdmin && claims.OrganizationID != nil {
		if site.OrganizationID != *claims.OrganizationID {
			Forbidden(c, "Cannot update a work site from a different organization")
			return
		}
	}

	var req struct {
		Name        string `json:"name"`
		ProjectRef  string `json:"project_ref"`
		Address     string `json:"address"`
		Description string `json:"description"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if req.Name != "" {
		site.Name = req.Name
	}
	site.ProjectRef = req.ProjectRef
	site.Address = req.Address
	site.Description = req.Description
	if req.IsActive != nil {
		site.IsActive = *req.IsActive
	}

	if err := h.repo.Update(c.Request.Context(), site); err != nil {
		InternalError(c, "Failed to update work site")
		return
	}
	SuccessResponse(c, http.StatusOK, toWorkSiteResponse(site))
}

// Delete godoc
// @Summary Delete a work site
// @Tags WorkSites
// @Param id path string true "Work Site ID"
// @Success 204
// @Router /api/v1/work-sites/{id} [delete]
func (h *WorkSiteHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid work site ID")
		return
	}

	// Ownership check for SACCO_ADMIN
	claims := middleware.GetClaims(c)
	if claims.SystemRole == types.RoleSaccoAdmin && claims.OrganizationID != nil {
		site, err := h.repo.GetByID(c.Request.Context(), id)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				NotFound(c, "Work site not found")
			} else {
				InternalError(c, "Failed to verify work site ownership")
			}
			return
		}
		if site.OrganizationID != *claims.OrganizationID {
			Forbidden(c, "Cannot delete a work site from a different organization")
			return
		}
	}

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			NotFound(c, "Work site not found")
		} else {
			InternalError(c, "Failed to delete work site")
		}
		return
	}
	c.Status(http.StatusNoContent)
}

// --- DTO ---

type workSiteResponse struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	ProjectRef     string    `json:"project_ref,omitempty"`
	Address        string    `json:"address,omitempty"`
	Description    string    `json:"description,omitempty"`
	IsActive       bool      `json:"is_active"`
	CreatedByID    uuid.UUID `json:"created_by_id"`
	CreatedAt      string    `json:"created_at"`
	UpdatedAt      string    `json:"updated_at"`
}

func toWorkSiteResponse(s *models.WorkSite) workSiteResponse {
	return workSiteResponse{
		ID:             s.ID,
		OrganizationID: s.OrganizationID,
		Name:           s.Name,
		ProjectRef:     s.ProjectRef,
		Address:        s.Address,
		Description:    s.Description,
		IsActive:       s.IsActive,
		CreatedByID:    s.CreatedByID,
		CreatedAt:      s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

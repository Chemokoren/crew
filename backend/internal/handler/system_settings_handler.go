package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// SystemSettingsHandler manages platform-wide settings, announcements, and maintenance mode.
type SystemSettingsHandler struct {
	settingsRepo     repository.SystemSettingRepository
	announcementRepo repository.SystemAnnouncementRepository
	statuteRepo      repository.StatutoryRateRepository
}

// NewSystemSettingsHandler creates a new SystemSettingsHandler.
func NewSystemSettingsHandler(
	settingsRepo repository.SystemSettingRepository,
	announcementRepo repository.SystemAnnouncementRepository,
	statuteRepo repository.StatutoryRateRepository,
) *SystemSettingsHandler {
	return &SystemSettingsHandler{
		settingsRepo:     settingsRepo,
		announcementRepo: announcementRepo,
		statuteRepo:      statuteRepo,
	}
}

// SystemStatus returns the current platform status (maintenance mode, etc.)
// This endpoint is public and does NOT require authentication.
func (h *SystemSettingsHandler) SystemStatus(c *gin.Context) {
	active := false
	message := ""

	if setting, err := h.settingsRepo.Get(c.Request.Context(), "maintenance.active"); err == nil && setting != nil {
		active = setting.Value == "true"
	}
	if active {
		if msgSetting, err := h.settingsRepo.Get(c.Request.Context(), "maintenance.message"); err == nil && msgSetting != nil {
			message = msgSetting.Value
		}
	}

	SuccessResponse(c, http.StatusOK, gin.H{
		"maintenance": active,
		"message":     message,
	})
}

// --- Statutory Rates ---

// CreateStatutoryRate godoc
// @Summary Create a statutory rate
// @Description Creates a new statutory deduction rate (NSSF, SHA, Housing Levy)
// @Tags SystemSettings
// @Accept json
// @Produce json
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/admin/statutory-rates [post]
func (h *SystemSettingsHandler) CreateStatutoryRate(c *gin.Context) {
	var rate models.StatutoryRate
	if err := c.ShouldBindJSON(&rate); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if err := h.statuteRepo.Create(c.Request.Context(), &rate); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, rate)
}

// UpdateStatutoryRate godoc
// @Summary Update a statutory rate
// @Description Updates an existing statutory deduction rate
// @Tags SystemSettings
// @Accept json
// @Produce json
// @Param id path string true "Rate ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/statutory-rates/{id} [put]
func (h *SystemSettingsHandler) UpdateStatutoryRate(c *gin.Context) {
	var rate models.StatutoryRate
	if err := c.ShouldBindJSON(&rate); err != nil {
		BadRequest(c, err.Error())
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid rate ID")
		return
	}
	rate.ID = id
	if err := h.statuteRepo.Update(c.Request.Context(), &rate); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, rate)
}

// --- System Settings (Key-Value) ---

// ListSettings godoc
// @Summary List all system settings
// @Description Returns all platform-level key-value settings
// @Tags SystemSettings
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/system-settings [get]
func (h *SystemSettingsHandler) ListSettings(c *gin.Context) {
	prefix := c.Query("prefix")
	var settings []models.SystemSetting
	var err error
	if prefix != "" {
		settings, err = h.settingsRepo.GetByPrefix(c.Request.Context(), prefix)
	} else {
		settings, err = h.settingsRepo.GetAll(c.Request.Context())
	}
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, settings)
}

// UpsertSetting godoc
// @Summary Create or update a system setting
// @Description Upserts a platform-level key-value setting
// @Tags SystemSettings
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/system-settings [put]
func (h *SystemSettingsHandler) UpsertSetting(c *gin.Context) {
	var setting models.SystemSetting
	if err := c.ShouldBindJSON(&setting); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if setting.Key == "" {
		BadRequest(c, "Setting key is required")
		return
	}
	if err := h.settingsRepo.Set(c.Request.Context(), &setting); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, setting)
}

// BulkUpsertSettings godoc
// @Summary Bulk create/update system settings
// @Description Upserts multiple settings at once
// @Tags SystemSettings
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/system-settings/bulk [put]
func (h *SystemSettingsHandler) BulkUpsertSettings(c *gin.Context) {
	var req struct {
		Settings []models.SystemSetting `json:"settings" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if err := h.settingsRepo.BulkSet(c.Request.Context(), req.Settings); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, gin.H{"message": "Settings saved", "count": len(req.Settings)})
}

// DeleteSetting godoc
// @Summary Delete a system setting
// @Description Deletes a platform-level key-value setting
// @Tags SystemSettings
// @Produce json
// @Param key path string true "Setting key"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/system-settings/{key} [delete]
func (h *SystemSettingsHandler) DeleteSetting(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		BadRequest(c, "Setting key is required")
		return
	}
	if err := h.settingsRepo.Delete(c.Request.Context(), key); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, gin.H{"message": "Setting deleted"})
}

// --- System Announcements ---

// ListAnnouncements godoc
// @Summary List all system announcements
// @Description Returns all platform-wide announcements (paginated)
// @Tags SystemSettings
// @Produce json
// @Param page query int false "Page number"
// @Param per_page query int false "Items per page"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/announcements [get]
func (h *SystemSettingsHandler) ListAnnouncements(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	announcements, total, err := h.announcementRepo.ListAll(c.Request.Context(), page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	ListResponse(c, announcements, buildMeta(page, perPage, total))
}

// ListActiveAnnouncements godoc
// @Summary List active system announcements
// @Description Returns currently active announcements visible to all users
// @Tags SystemSettings
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/announcements/active [get]
func (h *SystemSettingsHandler) ListActiveAnnouncements(c *gin.Context) {
	announcements, err := h.announcementRepo.ListActive(c.Request.Context())
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, announcements)
}

// CreateAnnouncement godoc
// @Summary Create a system announcement
// @Description Creates a new platform-wide announcement
// @Tags SystemSettings
// @Accept json
// @Produce json
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/admin/announcements [post]
func (h *SystemSettingsHandler) CreateAnnouncement(c *gin.Context) {
	var a models.SystemAnnouncement
	if err := c.ShouldBindJSON(&a); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if err := h.announcementRepo.Create(c.Request.Context(), &a); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, a)
}

// UpdateAnnouncement godoc
// @Summary Update a system announcement
// @Description Updates an existing announcement
// @Tags SystemSettings
// @Accept json
// @Produce json
// @Param id path string true "Announcement ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/announcements/{id} [put]
func (h *SystemSettingsHandler) UpdateAnnouncement(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid announcement ID")
		return
	}
	var a models.SystemAnnouncement
	if err := c.ShouldBindJSON(&a); err != nil {
		BadRequest(c, err.Error())
		return
	}
	a.ID = id
	if err := h.announcementRepo.Update(c.Request.Context(), &a); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, a)
}

// DeleteAnnouncement godoc
// @Summary Delete a system announcement
// @Description Deletes a system announcement
// @Tags SystemSettings
// @Produce json
// @Param id path string true "Announcement ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/announcements/{id} [delete]
func (h *SystemSettingsHandler) DeleteAnnouncement(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid announcement ID")
		return
	}
	if err := h.announcementRepo.Delete(c.Request.Context(), id); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, gin.H{"message": "Announcement deleted"})
}

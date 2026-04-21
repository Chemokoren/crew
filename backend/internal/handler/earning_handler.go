package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/repository"
)

type EarningHandler struct {
	earningRepo repository.EarningRepository
}

func NewEarningHandler(repo repository.EarningRepository) *EarningHandler {
	return &EarningHandler{earningRepo: repo}
}

func (h *EarningHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	var filter repository.EarningFilter
	if cm := c.Query("crew_member_id"); cm != "" {
		if id, err := uuid.Parse(cm); err == nil {
			filter.CrewMemberID = &id
		}
	}
	if a := c.Query("assignment_id"); a != "" {
		if id, err := uuid.Parse(a); err == nil {
			filter.AssignmentID = &id
		}
	}
	if df := c.Query("date_from"); df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			filter.DateFrom = &t
		}
	}
	if dt := c.Query("date_to"); dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			filter.DateTo = &t
		}
	}

	earnings, total, err := h.earningRepo.List(c.Request.Context(), filter, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	ListResponse(c, earnings, buildMeta(page, perPage, total))
}

func (h *EarningHandler) SummaryDashboard(c *gin.Context) {
	id, err := uuid.Parse(c.Param("crew_member_id"))
	if err != nil {
		BadRequest(c, "Invalid crew member ID")
		return
	}

	dateStr := c.Query("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		BadRequest(c, "Invalid date format, use YYYY-MM-DD")
		return
	}

	summary, err := h.earningRepo.GetDailySummary(c.Request.Context(), id, date)
	if err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, summary)
}

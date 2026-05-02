package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kibsoft/amy-mis/internal/ussd"
)

// USSDHandler handles USSD callback requests from Africa's Talking.
type USSDHandler struct {
	session *ussd.SessionHandler
}

// NewUSSDHandler creates a new USSD HTTP handler.
func NewUSSDHandler(session *ussd.SessionHandler) *USSDHandler {
	return &USSDHandler{session: session}
}

// Callback handles Africa's Talking USSD callbacks.
// AT sends form-encoded data: sessionId, phoneNumber, serviceCode, text.
// Response is plain text: "CON <menu>" to continue or "END <message>" to terminate.
//
// POST /api/v1/ussd/callback
func (h *USSDHandler) Callback(c *gin.Context) {
	var req ussd.USSDRequest
	if err := c.ShouldBind(&req); err != nil {
		c.String(http.StatusBadRequest, "END Invalid request")
		return
	}

	response := h.session.HandleSession(c.Request.Context(), req)
	c.String(http.StatusOK, response)
}

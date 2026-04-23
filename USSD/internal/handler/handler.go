// Package handler contains the HTTP request handler for USSD gateway endpoints.
// This is the thin controller layer — all business logic lives in the engine.
package handler

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/kibsoft/amy-mis-ussd/internal/engine"
	"github.com/kibsoft/amy-mis-ussd/internal/gateway"
	"github.com/kibsoft/amy-mis-ussd/internal/metrics"
	"github.com/kibsoft/amy-mis-ussd/internal/middleware"
	"github.com/kibsoft/amy-mis-ussd/internal/session"
)

// USSDHandler processes incoming USSD requests from telco gateways.
type USSDHandler struct {
	engine       *engine.Engine
	sessionStore *session.Store
	redis        *redis.Client
	logger       *slog.Logger
	defaultLang  string
}

// NewUSSDHandler creates a new USSD request handler.
func NewUSSDHandler(
	eng *engine.Engine,
	store *session.Store,
	redisClient *redis.Client,
	logger *slog.Logger,
	defaultLang string,
) *USSDHandler {
	return &USSDHandler{
		engine:       eng,
		sessionStore: store,
		redis:        redisClient,
		logger:       logger,
		defaultLang:  defaultLang,
	}
}

// Handle processes a USSD request from a specific gateway adapter.
func (h *USSDHandler) Handle(gw gateway.Gateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 1. Parse request
		req, err := gw.ParseRequest(c)
		if err != nil {
			h.logger.Error("failed to parse USSD request",
				slog.String("gateway", gw.Name()),
				slog.String("error", err.Error()),
			)
			metrics.USSDErrorsTotal.WithLabelValues("parse_error").Inc()
			gw.SendResponse(c, "Service error. Please try again.", true)
			return
		}

		// Store session ID in context for idempotency middleware
		c.Set("session_id", req.SessionID)

		h.logger.Debug("USSD request received",
			slog.String("session_id", req.SessionID),
			slog.String("msisdn", req.MSISDN),
			slog.String("input", req.Input),
			slog.String("gateway", gw.Name()),
		)

		// 2. Load or create session
		ctx := c.Request.Context()
		sess, err := h.sessionStore.Get(ctx, req.SessionID)
		if err != nil {
			h.logger.Error("session store error",
				slog.String("session_id", req.SessionID),
				slog.String("error", err.Error()),
			)
			metrics.USSDErrorsTotal.WithLabelValues("session_error").Inc()
			gw.SendResponse(c, "Service temporarily unavailable. Please try again.", true)
			return
		}

		isNewSession := false
		if sess == nil {
			// New session
			isNewSession = true
			sess = &session.Data{
				SessionID:    req.SessionID,
				MSISDN:       req.MSISDN,
				ServiceCode:  req.ServiceCode,
				CurrentState: session.StateInit,
				Language:     h.defaultLang,
				CreatedAt:    time.Now(),
				LastInputAt:  time.Now(),
			}
			metrics.USSDSessionsCreated.Inc()
			metrics.USSDSessionsActive.Inc()

			h.logger.Info("new USSD session",
				slog.String("session_id", req.SessionID),
				slog.String("msisdn", req.MSISDN),
			)
		} else {
			h.logger.Debug("existing session loaded",
				slog.String("session_id", req.SessionID),
				slog.String("state", string(sess.CurrentState)),
				slog.Int("step", sess.StepCount),
			)
		}

		// 3. Process through FSM engine
		resp, err := h.engine.Process(ctx, sess, req.Input)
		if err != nil {
			h.logger.Error("engine processing error",
				slog.String("session_id", req.SessionID),
				slog.String("state", string(sess.CurrentState)),
				slog.String("error", err.Error()),
			)
			metrics.USSDErrorsTotal.WithLabelValues("engine_error").Inc()
			gw.SendResponse(c, "Something went wrong. Please try again.", true)
			return
		}

		// 4. Persist session state
		if resp.EndSession {
			// Session completed — clean up
			if err := h.sessionStore.Delete(ctx, req.SessionID); err != nil {
				h.logger.Error("failed to delete session",
					slog.String("session_id", req.SessionID),
					slog.String("error", err.Error()),
				)
			}
			metrics.USSDSessionsCompleted.Inc()
			if !isNewSession {
				metrics.USSDSessionsActive.Dec()
			}
		} else {
			// Session continues — persist updated state
			if err := h.sessionStore.Save(ctx, sess); err != nil {
				h.logger.Error("failed to save session",
					slog.String("session_id", req.SessionID),
					slog.String("error", err.Error()),
				)
				metrics.USSDErrorsTotal.WithLabelValues("session_error").Inc()
			}
		}

		// 5. Send response
		responseText := formatUSSDResponse(resp)
		gw.SendResponse(c, resp.Message, resp.EndSession)

		// Cache response for idempotency
		middleware.CacheResponse(c, h.redis, responseText)

		// Log processing time
		duration := time.Since(start)
		h.logger.Info("USSD request processed",
			slog.String("session_id", req.SessionID),
			slog.String("state", string(sess.CurrentState)),
			slog.Bool("end_session", resp.EndSession),
			slog.Duration("latency", duration),
			slog.Int("step", sess.StepCount),
		)

		// Warn on slow responses
		if duration > 2*time.Second {
			h.logger.Warn("USSD response exceeded 2s threshold",
				slog.String("session_id", req.SessionID),
				slog.Duration("latency", duration),
			)
		}
	}
}

// formatUSSDResponse builds the full response string for caching.
func formatUSSDResponse(resp *engine.Response) string {
	prefix := "CON "
	if resp.EndSession {
		prefix = "END "
	}
	return prefix + resp.Message
}

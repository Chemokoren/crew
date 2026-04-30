// Package gateway provides a provider-agnostic abstraction for telco USSD gateways.
//
// Architecture: Strategy Pattern with Registry
//
//   ┌─────────────────────────────────────────────────────┐
//   │                   GatewayRegistry                    │
//   │  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │
//   │  │  AT      │  │ Generic  │  │ Future providers  │  │
//   │  │ (primary)│  │(fallback)│  │ (Twilio, Nexmo…)  │  │
//   │  └────┬─────┘  └────┬─────┘  └────────┬─────────┘  │
//   │       │             │                  │            │
//   │       └──────┬──────┘──────────────────┘            │
//   │              ▼                                      │
//   │        Gateway interface                            │
//   │     Primary() → selected adapter                    │
//   └─────────────────────────────────────────────────────┘
//
// All provider-specific details (AT's form-encoded body, cumulative text, CON/END
// prefixes) are encapsulated inside individual adapters. The handler and engine
// never know which telco is being used.
//
// Adding a new provider:
//   1. Implement the Gateway interface
//   2. Register it in the Registry with a unique name
//   3. Set PRIMARY_GATEWAY=<name> in .env — zero code changes needed
package gateway

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// --- Gateway Interface ---

// Request represents a normalized USSD request from any telco gateway.
// This is the common language between all gateway adapters and the FSM engine.
type Request struct {
	SessionID   string `json:"session_id"`
	MSISDN      string `json:"msisdn"`        // Phone number in international format (+254...)
	ServiceCode string `json:"service_code"`   // USSD shortcode e.g. *384*123#
	Input       string `json:"input"`          // User's current input
	NetworkCode string `json:"network_code"`   // MNO code (optional)
	Timestamp   time.Time
}

// Gateway defines the interface for telco USSD gateway adapters.
// Each telco provider implements this interface to normalize their
// proprietary format into our common Request/Response model.
type Gateway interface {
	// Name returns the gateway identifier (used for metrics and logging).
	Name() string

	// ParseRequest extracts a normalized USSD request from an HTTP request.
	ParseRequest(c *gin.Context) (*Request, error)

	// SendResponse formats and sends the USSD response back to the telco.
	SendResponse(c *gin.Context, message string, endSession bool)
}

// --- Gateway Registry (Strategy Pattern) ---

// Registry manages registered gateway adapters and provides strategy selection.
// Providers can be registered, enabled/disabled, and selected as primary or fallback
// at startup time — no code changes required to switch providers.
type Registry struct {
	gateways map[string]Gateway
	primary  string
	fallback string
	logger   *slog.Logger
}

// NewRegistry creates a new gateway registry.
func NewRegistry(logger *slog.Logger) *Registry {
	return &Registry{
		gateways: make(map[string]Gateway),
		logger:   logger,
	}
}

// Register adds a gateway adapter to the registry.
// Panics if a gateway with the same name is already registered.
func (r *Registry) Register(gw Gateway) {
	name := gw.Name()
	if _, exists := r.gateways[name]; exists {
		panic(fmt.Sprintf("gateway already registered: %s", name))
	}
	r.gateways[name] = gw
	r.logger.Info("gateway registered", slog.String("name", name))
}

// SetPrimary sets the primary gateway by name.
// Returns an error if the gateway is not registered.
func (r *Registry) SetPrimary(name string) error {
	if _, exists := r.gateways[name]; !exists {
		return fmt.Errorf("gateway not registered: %s (available: %s)", name, r.availableNames())
	}
	r.primary = name
	r.logger.Info("primary gateway set", slog.String("name", name))
	return nil
}

// SetFallback sets the fallback gateway by name.
// Returns an error if the gateway is not registered.
func (r *Registry) SetFallback(name string) error {
	if _, exists := r.gateways[name]; !exists {
		return fmt.Errorf("gateway not registered: %s (available: %s)", name, r.availableNames())
	}
	r.fallback = name
	r.logger.Info("fallback gateway set", slog.String("name", name))
	return nil
}

// Primary returns the primary gateway adapter.
// Falls back to the fallback gateway if the primary is not set.
func (r *Registry) Primary() Gateway {
	if gw, ok := r.gateways[r.primary]; ok {
		return gw
	}
	if gw, ok := r.gateways[r.fallback]; ok {
		r.logger.Warn("primary gateway unavailable, using fallback",
			slog.String("primary", r.primary),
			slog.String("fallback", r.fallback),
		)
		return gw
	}
	// Should never happen if config validation passes
	r.logger.Error("no gateway available", slog.String("primary", r.primary))
	return nil
}

// Fallback returns the fallback gateway adapter (used for simulator/dev).
func (r *Registry) Fallback() Gateway {
	if gw, ok := r.gateways[r.fallback]; ok {
		return gw
	}
	return nil
}

// Get returns a gateway by name, or nil if not registered.
func (r *Registry) Get(name string) Gateway {
	return r.gateways[name]
}

// Names returns the names of all registered gateways.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.gateways))
	for name := range r.gateways {
		names = append(names, name)
	}
	return names
}

// availableNames returns a comma-separated list of registered gateway names.
func (r *Registry) availableNames() string {
	return strings.Join(r.Names(), ", ")
}

// --- Africa's Talking Gateway ---

// AfricasTalkingGateway handles Africa's Talking USSD webhook format.
// AT sends POST with form-encoded body: sessionId, phoneNumber, serviceCode, text
type AfricasTalkingGateway struct {
	logger *slog.Logger
}

// NewAfricasTalkingGateway creates a new Africa's Talking gateway adapter.
func NewAfricasTalkingGateway(logger *slog.Logger) *AfricasTalkingGateway {
	return &AfricasTalkingGateway{logger: logger}
}

func (g *AfricasTalkingGateway) Name() string {
	return "africastalking"
}

// ParseRequest extracts the USSD request from Africa's Talking webhook format.
// AT sends the full conversation as `text` field with `*` delimiters.
// We extract only the latest input (last segment after the final `*`).
func (g *AfricasTalkingGateway) ParseRequest(c *gin.Context) (*Request, error) {
	sessionID := c.PostForm("sessionId")
	phoneNumber := c.PostForm("phoneNumber")
	serviceCode := c.PostForm("serviceCode")
	text := c.PostForm("text")
	networkCode := c.PostForm("networkCode")

	if sessionID == "" || phoneNumber == "" {
		return nil, fmt.Errorf("missing required fields: sessionId=%q, phoneNumber=%q", sessionID, phoneNumber)
	}

	// AT sends cumulative text: "1*2*3" — extract last input only
	input := extractLastInput(text)

	return &Request{
		SessionID:   sessionID,
		MSISDN:      normalizePhone(phoneNumber),
		ServiceCode: serviceCode,
		Input:       input,
		NetworkCode: networkCode,
		Timestamp:   time.Now(),
	}, nil
}

// SendResponse sends the USSD response in Africa's Talking format.
// CON = continue session, END = terminate session.
func (g *AfricasTalkingGateway) SendResponse(c *gin.Context, message string, endSession bool) {
	var prefix string
	if endSession {
		prefix = "END "
	} else {
		prefix = "CON "
	}

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, prefix+message)
}

// --- Generic/Simulator Gateway ---

// GenericGateway provides a JSON-based USSD interface for development and testing.
// This allows building USSD simulators that don't depend on telco-specific formats.
type GenericGateway struct {
	logger *slog.Logger
}

// NewGenericGateway creates a new generic/simulator gateway.
func NewGenericGateway(logger *slog.Logger) *GenericGateway {
	return &GenericGateway{logger: logger}
}

func (g *GenericGateway) Name() string {
	return "generic"
}

// genericRequest is the JSON request format for the generic gateway.
type genericRequest struct {
	SessionID   string `json:"session_id"`
	PhoneNumber string `json:"phone_number"`
	ServiceCode string `json:"service_code"`
	Input       string `json:"input"`
}

// ParseRequest extracts the USSD request from JSON body.
// Falls back to query parameters if the body is empty (convenient for
// browser/curl testing of the simulator endpoint).
func (g *GenericGateway) ParseRequest(c *gin.Context) (*Request, error) {
	var req genericRequest

	// Try JSON body first
	if err := c.ShouldBindJSON(&req); err != nil {
		// Fallback: read from query parameters (allows GET-style testing)
		req = genericRequest{
			SessionID:   c.Query("session_id"),
			PhoneNumber: c.Query("phone_number"),
			ServiceCode: c.Query("service_code"),
			Input:       c.Query("input"),
		}
	}

	if req.SessionID == "" || req.PhoneNumber == "" {
		return nil, fmt.Errorf("missing required fields: session_id and phone_number")
	}

	return &Request{
		SessionID:   req.SessionID,
		MSISDN:      normalizePhone(req.PhoneNumber),
		ServiceCode: req.ServiceCode,
		Input:       strings.TrimSpace(req.Input),
		Timestamp:   time.Now(),
	}, nil
}

// genericResponse is the JSON response format for the generic gateway.
type genericResponse struct {
	SessionID  string `json:"session_id"`
	Message    string `json:"message"`
	EndSession bool   `json:"end_session"`
}

// SendResponse sends the USSD response as JSON.
func (g *GenericGateway) SendResponse(c *gin.Context, message string, endSession bool) {
	resp := genericResponse{
		SessionID:  c.GetString("session_id"),
		Message:    message,
		EndSession: endSession,
	}

	c.JSON(http.StatusOK, resp)
}

// --- Helpers ---

// extractLastInput pulls the most recent user input from AT's cumulative text field.
// AT sends "1*2*3" for a session where user entered 1, then 2, then 3.
// We return "3" (the latest input).
func extractLastInput(text string) string {
	if text == "" {
		return ""
	}
	parts := strings.Split(text, "*")
	return strings.TrimSpace(parts[len(parts)-1])
}

// normalizePhone ensures phone numbers are in +254 format.
func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")

	if strings.HasPrefix(phone, "+") {
		return phone
	}
	if strings.HasPrefix(phone, "0") {
		return "+254" + phone[1:]
	}
	if strings.HasPrefix(phone, "254") {
		return "+" + phone
	}
	return phone
}

// MarshalRequest serializes a request to JSON (for logging/debugging).
func MarshalRequest(req *Request) string {
	b, _ := json.Marshal(req)
	return string(b)
}

// Package routing maps USSD service codes to industries with embedded job types.
//
// Two-level design:
//
//	Level 1 (Primary) — Industry-specific:
//	  Each service code maps to an industry. The industry's job types are
//	  hardcoded in this package — zero network dependency, zero failure modes.
//	  A construction user dialing *384*200# will ALWAYS see Mason/Carpenter/etc.
//
//	Level 2 (Secondary) — Tenant-specific override:
//	  A service code can optionally pin to a specific organization ID.
//	  If set, the engine fetches that org's custom job types from the backend.
//	  If the fetch fails, Level 1 (industry defaults) is used — still correct.
//
// Configuration via SERVICE_CODE_ROUTES environment variable:
//
//	*384*123#:TRANSPORT,*384*200#:CONSTRUCTION,*384*201#:CONSTRUCTION@<org-uuid>
package routing

import (
	"fmt"
	"strings"
)

// --- Embedded job type definitions (mirrors backend industry_templates.go) ---

// JobType holds a worker role for USSD registration menus.
type JobType struct {
	Code        string // Machine-readable code (e.g. MASON, CHV)
	DisplayName string // Human-readable label shown on USSD menu
}

// industryJobTypes contains the hardcoded, guaranteed-correct job types per industry.
// These NEVER depend on a network call. They mirror the backend's industry_templates.go
// but are locally owned so the USSD service can always render the correct menu.
//
// Only PRIMARY and FACILITATOR roles are included — SUPERVISOR and SUPPORT roles
// are admin-assigned, not self-registered.
var industryJobTypes = map[string][]JobType{
	"TRANSPORT": {
		{Code: "DRIVER", DisplayName: "Driver"},
		{Code: "CONDUCTOR", DisplayName: "Conductor"},
		{Code: "RIDER", DisplayName: "Boda Rider"},
		{Code: "BOOKING_AGENT", DisplayName: "Booking Agent"},
		{Code: "DISPATCHER", DisplayName: "Dispatcher"},
	},
	"CONSTRUCTION": {
		{Code: "MASON", DisplayName: "Mason"},
		{Code: "CARPENTER", DisplayName: "Carpenter"},
		{Code: "PLUMBER", DisplayName: "Plumber"},
		{Code: "ELECTRICIAN", DisplayName: "Electrician"},
		{Code: "LABORER", DisplayName: "General Laborer"},
	},
	"HEALTH": {
		{Code: "CHV", DisplayName: "Community Health Volunteer"},
		{Code: "CHP", DisplayName: "Community Health Promoter"},
		{Code: "NURSE", DisplayName: "Nurse"},
	},
	"LOGISTICS": {
		{Code: "RIDER", DisplayName: "Delivery Rider"},
		{Code: "DRIVER", DisplayName: "Driver"},
		{Code: "LOADER", DisplayName: "Loader"},
		{Code: "DISPATCHER", DisplayName: "Dispatcher"},
	},
	"AGRICULTURE": {
		{Code: "PICKER", DisplayName: "Picker / Harvester"},
		{Code: "FIELD_WORKER", DisplayName: "Field Worker"},
		{Code: "SORTER", DisplayName: "Sorter / Grader"},
	},
	"HOSPITALITY": {
		{Code: "WAITER", DisplayName: "Waiter / Waitress"},
		{Code: "COOK", DisplayName: "Cook"},
		{Code: "HOUSEKEEPER", DisplayName: "Housekeeper"},
	},
}

// GetIndustryJobTypes returns the hardcoded job types for an industry.
// Always returns a non-nil slice. Unknown industries get a generic fallback.
func GetIndustryJobTypes(industryType string) []JobType {
	if jts, ok := industryJobTypes[industryType]; ok {
		return jts
	}
	// Generic fallback — still valid, never wrong
	return []JobType{
		{Code: "WORKER", DisplayName: "Worker"},
	}
}

// --- Service code routing ---

// Route holds the resolved context for a USSD service code.
type Route struct {
	ServiceCode    string // The dialed shortcode
	IndustryType   string // TRANSPORT, CONSTRUCTION, HEALTH, etc.
	OrganizationID string // Optional: tenant override (Level 2)
}

// Table maps service codes to their routing context.
// Populated once at startup; read concurrently — no lock needed.
type Table struct {
	routes       map[string]Route
	defaultRoute Route
}

// NewTable creates a routing table from a comma-separated config string.
//
// Format per entry:  <service_code>:<INDUSTRY_TYPE>[@<org_id>]
//
// Examples:
//
//	*384*123#:TRANSPORT
//	*384*200#:CONSTRUCTION
//	*384*201#:CONSTRUCTION@550e8400-e29b-41d4-a716-446655440000
func NewTable(configStr string) *Table {
	t := &Table{
		routes: make(map[string]Route),
		defaultRoute: Route{
			IndustryType: "TRANSPORT", // Legacy default
		},
	}

	if configStr == "" {
		return t
	}

	for _, entry := range strings.Split(configStr, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			continue
		}

		code := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		route := Route{ServiceCode: code}

		// Check for tenant override: INDUSTRY@org_id
		if idx := strings.Index(value, "@"); idx >= 0 {
			route.IndustryType = value[:idx]
			route.OrganizationID = value[idx+1:]
		} else {
			route.IndustryType = value
		}

		t.routes[code] = route
	}

	return t
}

// Resolve returns the routing context for a service code.
// Falls back to TRANSPORT if the code is not mapped.
func (t *Table) Resolve(serviceCode string) Route {
	serviceCode = normalizeCode(serviceCode)
	if r, ok := t.routes[serviceCode]; ok {
		return r
	}
	return t.defaultRoute
}

// HasRoute checks if a specific service code has an explicit mapping.
func (t *Table) HasRoute(serviceCode string) bool {
	_, ok := t.routes[normalizeCode(serviceCode)]
	return ok
}

// AllRoutes returns all configured routes for cache refresh iteration.
func (t *Table) AllRoutes() []Route {
	routes := make([]Route, 0, len(t.routes))
	for _, r := range t.routes {
		routes = append(routes, r)
	}
	return routes
}

// String returns a human-readable representation of the routing table.
func (t *Table) String() string {
	if len(t.routes) == 0 {
		return "(default: TRANSPORT)"
	}
	var entries []string
	for code, r := range t.routes {
		s := fmt.Sprintf("%s→%s", code, r.IndustryType)
		if r.OrganizationID != "" {
			s += "@" + r.OrganizationID
		}
		entries = append(entries, s)
	}
	return strings.Join(entries, ", ")
}

// normalizeCode strips whitespace from the service code for matching.
func normalizeCode(code string) string {
	return strings.TrimSpace(code)
}

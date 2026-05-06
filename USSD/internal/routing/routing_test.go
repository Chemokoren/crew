package routing

import (
	"testing"
)

func TestNewTable_Empty(t *testing.T) {
	tbl := NewTable("")
	r := tbl.Resolve("*384*123#")
	if r.IndustryType != "TRANSPORT" {
		t.Errorf("empty config should default to TRANSPORT, got %q", r.IndustryType)
	}
}

func TestNewTable_SingleRoute(t *testing.T) {
	tbl := NewTable("*384*123#:TRANSPORT")
	r := tbl.Resolve("*384*123#")
	if r.IndustryType != "TRANSPORT" {
		t.Errorf("got %q, want TRANSPORT", r.IndustryType)
	}
	if r.OrganizationID != "" {
		t.Errorf("got org %q, want empty", r.OrganizationID)
	}
}

func TestNewTable_MultipleRoutes(t *testing.T) {
	tbl := NewTable("*384*123#:TRANSPORT, *384*200#:CONSTRUCTION, *384*300#:HEALTH")

	tests := []struct {
		code     string
		industry string
	}{
		{"*384*123#", "TRANSPORT"},
		{"*384*200#", "CONSTRUCTION"},
		{"*384*300#", "HEALTH"},
	}
	for _, tt := range tests {
		r := tbl.Resolve(tt.code)
		if r.IndustryType != tt.industry {
			t.Errorf("Resolve(%q).IndustryType = %q, want %q", tt.code, r.IndustryType, tt.industry)
		}
	}
}

func TestNewTable_TenantOverride(t *testing.T) {
	tbl := NewTable("*384*201#:CONSTRUCTION@org-uuid-123")
	r := tbl.Resolve("*384*201#")
	if r.IndustryType != "CONSTRUCTION" {
		t.Errorf("got industry %q, want CONSTRUCTION", r.IndustryType)
	}
	if r.OrganizationID != "org-uuid-123" {
		t.Errorf("got org %q, want org-uuid-123", r.OrganizationID)
	}
}

func TestNewTable_UnknownCodeFallsBack(t *testing.T) {
	tbl := NewTable("*384*123#:TRANSPORT")
	r := tbl.Resolve("*999*999#")
	if r.IndustryType != "TRANSPORT" {
		t.Errorf("unknown code should default to TRANSPORT, got %q", r.IndustryType)
	}
}

func TestHasRoute(t *testing.T) {
	tbl := NewTable("*384*123#:TRANSPORT")
	if !tbl.HasRoute("*384*123#") {
		t.Error("should have route for *384*123#")
	}
	if tbl.HasRoute("*999*999#") {
		t.Error("should not have route for *999*999#")
	}
}

// --- Embedded industry job types tests ---

func TestGetIndustryJobTypes_AllIndustries(t *testing.T) {
	industries := []struct {
		name     string
		minRoles int
	}{
		{"TRANSPORT", 3},
		{"CONSTRUCTION", 5},
		{"HEALTH", 3},
		{"LOGISTICS", 3},
		{"AGRICULTURE", 3},
		{"HOSPITALITY", 3},
	}

	for _, tt := range industries {
		t.Run(tt.name, func(t *testing.T) {
			jts := GetIndustryJobTypes(tt.name)
			if len(jts) < tt.minRoles {
				t.Errorf("GetIndustryJobTypes(%q) returned %d roles, want at least %d",
					tt.name, len(jts), tt.minRoles)
			}
			// Verify every entry has Code and DisplayName set
			for _, jt := range jts {
				if jt.Code == "" || jt.DisplayName == "" {
					t.Errorf("job type has empty field: Code=%q, DisplayName=%q", jt.Code, jt.DisplayName)
				}
			}
		})
	}
}

func TestGetIndustryJobTypes_Transport(t *testing.T) {
	jts := GetIndustryJobTypes("TRANSPORT")
	codes := make(map[string]bool)
	for _, jt := range jts {
		codes[jt.Code] = true
	}
	for _, want := range []string{"DRIVER", "CONDUCTOR", "RIDER"} {
		if !codes[want] {
			t.Errorf("TRANSPORT missing %q role", want)
		}
	}
}

func TestGetIndustryJobTypes_Construction(t *testing.T) {
	jts := GetIndustryJobTypes("CONSTRUCTION")
	codes := make(map[string]bool)
	for _, jt := range jts {
		codes[jt.Code] = true
	}
	for _, want := range []string{"MASON", "CARPENTER", "PLUMBER"} {
		if !codes[want] {
			t.Errorf("CONSTRUCTION missing %q role", want)
		}
	}
}

func TestGetIndustryJobTypes_UnknownIndustry(t *testing.T) {
	jts := GetIndustryJobTypes("UNKNOWN_INDUSTRY")
	if len(jts) == 0 {
		t.Error("unknown industry should return generic fallback, got empty")
	}
	if jts[0].Code != "WORKER" {
		t.Errorf("unknown industry fallback code = %q, want WORKER", jts[0].Code)
	}
}

func TestGetIndustryJobTypes_NoSupervisorRoles(t *testing.T) {
	// Self-registration menus should NOT contain SUPERVISOR or SUPPORT roles.
	// Verify none of the embedded lists include them.
	supervisorCodes := map[string]bool{
		"SUPERVISOR": true, "FOREMAN": true, "SITE_MANAGER": true,
		"SAFETY_OFFICER": true, "COORDINATOR": true, "DATA_CLERK": true,
		"OFFICE_ADMIN": true, "SHIFT_LEAD": true, "TEAM_LEAD": true,
		"RECEPTIONIST": true, "WEIGHBRIDGE": true, "WAREHOUSE_MGR": true,
	}
	for industry, jts := range industryJobTypes {
		for _, jt := range jts {
			if supervisorCodes[jt.Code] {
				t.Errorf("industry %q contains non-registerable role %q", industry, jt.Code)
			}
		}
	}
}

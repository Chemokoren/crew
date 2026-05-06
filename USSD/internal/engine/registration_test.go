package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kibsoft/amy-mis-ussd/internal/rolecache"
	"github.com/kibsoft/amy-mis-ussd/internal/routing"
	"github.com/kibsoft/amy-mis-ussd/internal/session"
)

// --- Mock infrastructure for integration tests ---

// mockRedisStore is an in-memory RedisStore for testing.
type mockRedisStore struct {
	data map[string][]byte
}

func newMockRedisStore() *mockRedisStore {
	return &mockRedisStore{data: make(map[string][]byte)}
}

func (m *mockRedisStore) GetBytes(_ context.Context, key string) ([]byte, error) {
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("key not found")
}

func (m *mockRedisStore) SetBytes(_ context.Context, key string, value []byte, _ time.Duration) error {
	m.data[key] = value
	return nil
}

// newTestSession creates a session.Data for testing registration flow.
func newTestSession(serviceCode, msisdn string) *session.Data {
	return &session.Data{
		SessionID:   "test-reg-001",
		MSISDN:      msisdn,
		ServiceCode: serviceCode,
		Language:    "en",
		StepCount:   1,
	}
}

// --- Integration Tests ---

// TestRegistrationFlow_TransportIndustry verifies that dialing *384*123# shows
// transport roles (Driver, Conductor, Boda Rider, Booking Agent).
func TestRegistrationFlow_TransportIndustry(t *testing.T) {
	routeTable := routing.NewTable("*384*123#:TRANSPORT")
	store := newMockRedisStore()
	roleCache := rolecache.NewCache(store, routeTable, nil, nil)

	sess := newTestSession("*384*123#", "+254700000001")
	sess.CurrentState = session.StateRegisterRole

	// Simulate: user enters national ID, engine calls buildRoleMenu
	roles := roleCache.GetRoles(context.Background(), "*384*123#")

	if len(roles) < 3 {
		t.Fatalf("expected at least 3 transport roles, got %d", len(roles))
	}

	codes := make(map[string]bool)
	for _, r := range roles {
		codes[r.Code] = true
	}

	for _, want := range []string{"DRIVER", "CONDUCTOR", "RIDER"} {
		if !codes[want] {
			t.Errorf("transport registration missing role %q", want)
		}
	}

	// Verify no supervisor/support roles leaked
	forbidden := []string{"SUPERVISOR", "OFFICE_ADMIN", "FOREMAN", "SAFETY_OFFICER"}
	for _, f := range forbidden {
		if codes[f] {
			t.Errorf("transport registration contains forbidden role %q", f)
		}
	}
}

// TestRegistrationFlow_ConstructionIndustry verifies that dialing *384*200# shows
// construction roles, NOT transport roles.
func TestRegistrationFlow_ConstructionIndustry(t *testing.T) {
	routeTable := routing.NewTable("*384*123#:TRANSPORT,*384*200#:CONSTRUCTION")
	store := newMockRedisStore()
	roleCache := rolecache.NewCache(store, routeTable, nil, nil)

	roles := roleCache.GetRoles(context.Background(), "*384*200#")

	codes := make(map[string]bool)
	for _, r := range roles {
		codes[r.Code] = true
	}

	// Must have construction roles
	for _, want := range []string{"MASON", "CARPENTER", "PLUMBER"} {
		if !codes[want] {
			t.Errorf("construction registration missing role %q", want)
		}
	}

	// Must NOT have transport roles
	for _, wrong := range []string{"DRIVER", "CONDUCTOR"} {
		if codes[wrong] {
			t.Errorf("construction registration incorrectly contains transport role %q", wrong)
		}
	}
}

// TestRegistrationFlow_TenantOverrideFromCache verifies that when a service code
// has a tenant override cached in Redis, those roles are served instead of industry defaults.
func TestRegistrationFlow_TenantOverrideFromCache(t *testing.T) {
	routeTable := routing.NewTable("*384*201#:CONSTRUCTION@org-tenant-123")
	store := newMockRedisStore()

	// Pre-populate cache with tenant-specific roles
	tenantRoles := []rolecache.CachedRole{
		{Code: "TILER", DisplayName: "Tiler", JobTypeID: "jt-t001"},
		{Code: "GLAZIER", DisplayName: "Glazier", JobTypeID: "jt-t002"},
		{Code: "ROOFER", DisplayName: "Roofer", JobTypeID: "jt-t003"},
	}
	data, _ := json.Marshal(tenantRoles)
	store.data["ussd:roles:*384*201#"] = data

	roleCache := rolecache.NewCache(store, routeTable, nil, nil)

	roles := roleCache.GetRoles(context.Background(), "*384*201#")

	if len(roles) != 3 {
		t.Fatalf("expected 3 tenant roles from cache, got %d", len(roles))
	}

	// Verify tenant roles, not generic construction
	if roles[0].Code != "TILER" {
		t.Errorf("expected TILER, got %q", roles[0].Code)
	}
	if roles[0].JobTypeID != "jt-t001" {
		t.Errorf("expected job type ID jt-t001, got %q", roles[0].JobTypeID)
	}

	// Verify generic construction roles are NOT returned
	for _, r := range roles {
		if r.Code == "MASON" || r.Code == "CARPENTER" {
			t.Errorf("got generic industry role %q, expected tenant-specific", r.Code)
		}
	}
}

// TestRegistrationFlow_TenantCacheMiss_FallsToIndustry verifies that when a
// tenant service code has no cache entry, the industry defaults are used.
func TestRegistrationFlow_TenantCacheMiss_FallsToIndustry(t *testing.T) {
	routeTable := routing.NewTable("*384*201#:CONSTRUCTION@org-tenant-123")
	store := newMockRedisStore() // Empty — no cache entry

	roleCache := rolecache.NewCache(store, routeTable, nil, nil)

	roles := roleCache.GetRoles(context.Background(), "*384*201#")

	// Should get construction defaults, not empty
	if len(roles) == 0 {
		t.Fatal("expected construction fallback roles, got none")
	}

	codes := make(map[string]bool)
	for _, r := range roles {
		codes[r.Code] = true
	}

	if !codes["MASON"] {
		t.Error("tenant cache miss should fall back to industry defaults, missing MASON")
	}
}

// TestRegistrationFlow_HealthIndustry verifies health service code routing.
func TestRegistrationFlow_HealthIndustry(t *testing.T) {
	routeTable := routing.NewTable("*384*300#:HEALTH")
	store := newMockRedisStore()
	roleCache := rolecache.NewCache(store, routeTable, nil, nil)

	roles := roleCache.GetRoles(context.Background(), "*384*300#")

	codes := make(map[string]bool)
	for _, r := range roles {
		codes[r.Code] = true
	}

	for _, want := range []string{"CHV", "CHP", "NURSE"} {
		if !codes[want] {
			t.Errorf("health registration missing role %q", want)
		}
	}

	// Must NOT contain transport or construction roles
	for _, wrong := range []string{"DRIVER", "MASON"} {
		if codes[wrong] {
			t.Errorf("health registration incorrectly contains non-health role %q", wrong)
		}
	}
}

// TestRegistrationFlow_RoleMenuRendering verifies that the role menu is correctly
// rendered as numbered USSD text from cached roles.
func TestRegistrationFlow_RoleMenuRendering(t *testing.T) {
	routeTable := routing.NewTable("*384*200#:CONSTRUCTION")
	store := newMockRedisStore()
	roleCache := rolecache.NewCache(store, routeTable, nil, nil)

	sess := newTestSession("*384*200#", "+254700000002")

	roles := roleCache.GetRoles(context.Background(), "*384*200#")

	// Simulate buildRoleMenu logic
	menu := "Select your role"
	for i, role := range roles {
		idx := i + 1
		menu += fmt.Sprintf("\n%d. %s", idx, role.DisplayName)
		sess.SetInput(fmt.Sprintf("jt_%d_code", idx), role.Code)
		sess.SetInput(fmt.Sprintf("jt_%d_name", idx), role.DisplayName)
	}
	menu += "\n0. Back"

	// Verify menu structure
	if !strings.Contains(menu, "1. Mason") {
		t.Errorf("menu missing '1. Mason', got:\n%s", menu)
	}
	if !strings.Contains(menu, "0. Back") {
		t.Errorf("menu missing '0. Back'")
	}

	// Verify session state for role selection
	if sess.GetInput("jt_1_code") != "MASON" {
		t.Errorf("session jt_1_code = %q, want MASON", sess.GetInput("jt_1_code"))
	}
	if sess.GetInput("jt_2_code") != "CARPENTER" {
		t.Errorf("session jt_2_code = %q, want CARPENTER", sess.GetInput("jt_2_code"))
	}
}

// TestRegistrationFlow_AllIndustries_NeverEmpty verifies that every configured
// industry produces a non-empty role menu.
func TestRegistrationFlow_AllIndustries_NeverEmpty(t *testing.T) {
	industries := map[string]string{
		"*384*123#": "TRANSPORT",
		"*384*200#": "CONSTRUCTION",
		"*384*300#": "HEALTH",
		"*384*400#": "LOGISTICS",
		"*384*500#": "AGRICULTURE",
		"*384*600#": "HOSPITALITY",
	}

	routeConfig := ""
	for code, industry := range industries {
		if routeConfig != "" {
			routeConfig += ","
		}
		routeConfig += code + ":" + industry
	}

	routeTable := routing.NewTable(routeConfig)
	store := newMockRedisStore()
	roleCache := rolecache.NewCache(store, routeTable, nil, nil)

	for code, industry := range industries {
		t.Run(industry, func(t *testing.T) {
			roles := roleCache.GetRoles(context.Background(), code)
			if len(roles) == 0 {
				t.Errorf("%s (%s): registration menu would be empty — this is a critical failure", industry, code)
			}
			for _, r := range roles {
				if r.Code == "" || r.DisplayName == "" {
					t.Errorf("%s: role has empty field: Code=%q DisplayName=%q", industry, r.Code, r.DisplayName)
				}
			}
		})
	}
}

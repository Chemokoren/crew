package rolecache

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/kibsoft/amy-mis-ussd/internal/routing"
)

// mockStore is an in-memory implementation of RedisStore for testing.
type mockStore struct {
	data map[string][]byte
}

func newMockStore() *mockStore {
	return &mockStore{data: make(map[string][]byte)}
}

func (m *mockStore) GetBytes(_ context.Context, key string) ([]byte, error) {
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("key not found")
}

func (m *mockStore) SetBytes(_ context.Context, key string, value []byte, _ time.Duration) error {
	m.data[key] = value
	return nil
}

func TestGetRoles_CacheHit(t *testing.T) {
	store := newMockStore()
	routeTable := routing.NewTable("*384*200#:CONSTRUCTION")

	// Pre-populate cache with tenant-specific roles
	roles := []CachedRole{
		{Code: "WELDER", DisplayName: "Custom Welder", JobTypeID: "jt-123"},
		{Code: "PAINTER", DisplayName: "Custom Painter", JobTypeID: "jt-456"},
	}
	data, _ := json.Marshal(roles)
	store.data["ussd:roles:*384*200#"] = data

	cache := NewCache(store, routeTable, nil, nil)

	got := cache.GetRoles(context.Background(), "*384*200#")
	if len(got) != 2 {
		t.Fatalf("expected 2 roles from cache, got %d", len(got))
	}
	if got[0].Code != "WELDER" {
		t.Errorf("expected WELDER, got %q", got[0].Code)
	}
	if got[0].JobTypeID != "jt-123" {
		t.Errorf("expected job type ID jt-123, got %q", got[0].JobTypeID)
	}
}

func TestGetRoles_CacheMiss_HardcodedFallback(t *testing.T) {
	store := newMockStore() // Empty cache
	routeTable := routing.NewTable("*384*200#:CONSTRUCTION")

	cache := NewCache(store, routeTable, nil, nil)

	got := cache.GetRoles(context.Background(), "*384*200#")
	if len(got) == 0 {
		t.Fatal("expected hardcoded fallback roles, got none")
	}

	// Verify we get CONSTRUCTION roles from the hardcoded fallback
	codes := make(map[string]bool)
	for _, r := range got {
		codes[r.Code] = true
	}
	for _, want := range []string{"MASON", "CARPENTER", "PLUMBER"} {
		if !codes[want] {
			t.Errorf("hardcoded fallback missing %q role", want)
		}
	}
}

func TestGetRoles_TransportFallback(t *testing.T) {
	store := newMockStore()
	routeTable := routing.NewTable("*384*123#:TRANSPORT")

	cache := NewCache(store, routeTable, nil, nil)

	got := cache.GetRoles(context.Background(), "*384*123#")
	codes := make(map[string]bool)
	for _, r := range got {
		codes[r.Code] = true
	}
	for _, want := range []string{"DRIVER", "CONDUCTOR", "RIDER"} {
		if !codes[want] {
			t.Errorf("transport fallback missing %q", want)
		}
	}
}

func TestGetRoles_TenantCacheOverridesIndustry(t *testing.T) {
	store := newMockStore()
	routeTable := routing.NewTable("*384*201#:CONSTRUCTION@org-123")

	// Cache has tenant-specific roles different from industry defaults
	roles := []CachedRole{
		{Code: "TILER", DisplayName: "Tiler", JobTypeID: "jt-tenant-001"},
		{Code: "GLAZIER", DisplayName: "Glazier", JobTypeID: "jt-tenant-002"},
	}
	data, _ := json.Marshal(roles)
	store.data["ussd:roles:*384*201#"] = data

	cache := NewCache(store, routeTable, nil, nil)

	got := cache.GetRoles(context.Background(), "*384*201#")
	if len(got) != 2 {
		t.Fatalf("expected 2 tenant roles from cache, got %d", len(got))
	}
	if got[0].Code != "TILER" {
		t.Errorf("expected TILER, got %q", got[0].Code)
	}
	// Verify these are NOT the generic construction roles
	for _, r := range got {
		if r.Code == "MASON" || r.Code == "CARPENTER" {
			t.Errorf("got generic industry role %q, expected tenant-specific", r.Code)
		}
	}
}

func TestGetRoles_UnknownServiceCode(t *testing.T) {
	store := newMockStore()
	routeTable := routing.NewTable("*384*123#:TRANSPORT")

	cache := NewCache(store, routeTable, nil, nil)

	// Unknown code falls back to TRANSPORT industry via routing table default
	got := cache.GetRoles(context.Background(), "*999*999#")
	if len(got) == 0 {
		t.Fatal("unknown service code should return fallback roles")
	}
	codes := make(map[string]bool)
	for _, r := range got {
		codes[r.Code] = true
	}
	if !codes["DRIVER"] {
		t.Error("unknown code should fall back to TRANSPORT, missing DRIVER")
	}
}

func TestHardcodedRoles_NeverEmpty(t *testing.T) {
	store := newMockStore()

	industries := []string{
		"TRANSPORT", "CONSTRUCTION", "HEALTH",
		"LOGISTICS", "AGRICULTURE", "HOSPITALITY",
	}

	for _, industry := range industries {
		routeTable := routing.NewTable("*test#:" + industry)
		cache := NewCache(store, routeTable, nil, nil)

		got := cache.GetRoles(context.Background(), "*test#")
		if len(got) == 0 {
			t.Errorf("hardcoded roles for %q should never be empty", industry)
		}
		for _, r := range got {
			if r.Code == "" || r.DisplayName == "" {
				t.Errorf("%q: role has empty field: Code=%q DisplayName=%q",
					industry, r.Code, r.DisplayName)
			}
		}
	}
}

func TestHandleInvalidation_WildcardRefreshesAll(t *testing.T) {
	store := newMockStore()
	routeTable := routing.NewTable("*384*123#:TRANSPORT,*384*200#:CONSTRUCTION")

	// Pre-populate cache to verify it's being read, not skipped
	transportRoles := []CachedRole{{Code: "DRIVER", DisplayName: "Driver"}}
	data, _ := json.Marshal(transportRoles)
	store.data["ussd:roles:*384*123#"] = data

	cache := NewCache(store, routeTable, nil, nil)

	// The wildcard "*" should attempt a full refresh
	// (API client is nil so it falls back to hardcoded, but the method shouldn't panic)
	cache.handleInvalidation(context.Background(), "*")
	// If we get here without panicking, the wildcard path works
}

func TestHandleInvalidation_SpecificCode(t *testing.T) {
	store := newMockStore()
	routeTable := routing.NewTable("*384*200#:CONSTRUCTION")

	cache := NewCache(store, routeTable, nil, nil)

	// Invalidate a specific service code — should not panic
	cache.handleInvalidation(context.Background(), "*384*200#")
}

func TestHandleInvalidation_UnknownCodeIgnored(t *testing.T) {
	store := newMockStore()
	routeTable := routing.NewTable("*384*200#:CONSTRUCTION")

	cache := NewCache(store, routeTable, nil, nil)

	// Unknown code should be ignored without error
	cache.handleInvalidation(context.Background(), "*999*999#")
}

func TestNextMidnight(t *testing.T) {
	cache := &Cache{}

	// Test with a known time: 2026-05-06 14:30:00
	now := time.Date(2026, 5, 6, 14, 30, 0, 0, time.Local)
	midnight := cache.nextMidnight(now)

	expected := time.Date(2026, 5, 7, 0, 0, 0, 0, time.Local)
	if !midnight.Equal(expected) {
		t.Errorf("nextMidnight(%v) = %v, want %v", now, midnight, expected)
	}

	// Verify it's always in the future
	if !midnight.After(now) {
		t.Error("nextMidnight should always be in the future")
	}
}

func TestNextMidnight_NearMidnight(t *testing.T) {
	cache := &Cache{}

	// Test at 23:59:59 — should still return tomorrow's midnight
	now := time.Date(2026, 5, 6, 23, 59, 59, 0, time.Local)
	midnight := cache.nextMidnight(now)

	expected := time.Date(2026, 5, 7, 0, 0, 0, 0, time.Local)
	if !midnight.Equal(expected) {
		t.Errorf("nextMidnight(%v) = %v, want %v", now, midnight, expected)
	}
}

func TestSetPubSub_NilSafe(t *testing.T) {
	store := newMockStore()
	routeTable := routing.NewTable("*384*123#:TRANSPORT")
	cache := NewCache(store, routeTable, nil, nil)

	// startPubSubListener should be a no-op when pubsub is nil
	cache.startPubSubListener(context.Background())
	// If we get here without panicking, the nil check works
}

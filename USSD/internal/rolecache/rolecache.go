// Package rolecache provides a 3-layer cache for USSD registration roles.
//
// Architecture:
//
//	Layer 1 (Hot)  — Redis cache: sub-millisecond reads, populated by background cron.
//	Layer 2 (Cold) — Hardcoded roles in routing package: compiled into binary, never fails.
//	Layer 3 (Background) — API fetch via midnight cron: keeps Redis warm with latest
//	                        industry templates and tenant-specific overrides.
//
// The USSD user experience is NEVER affected by API latency:
//   - GetRoles() reads from Redis → falls back to hardcoded → never blocks on API.
//   - The background cron refreshes Redis asynchronously at configurable intervals.
//   - On startup, an initial population attempt is made best-effort.
//   - Redis Pub/Sub provides event-driven invalidation for immediate updates.
//
// Cache key format:  ussd:roles:<normalized_service_code>
// Cache value:       JSON array of [{code, display_name, job_type_id}]
// Pub/Sub channel:   ussd:role_cache:invalidate (payload: service_code or "*" for all)
package rolecache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/kibsoft/amy-mis-ussd/internal/backend"
	"github.com/kibsoft/amy-mis-ussd/internal/metrics"
	"github.com/kibsoft/amy-mis-ussd/internal/routing"
)

// RedisStore is the minimal interface the role cache needs from Redis.
// *redis.Client satisfies this naturally; tests can provide a simple mock.
type RedisStore interface {
	GetBytes(ctx context.Context, key string) ([]byte, error)
	SetBytes(ctx context.Context, key string, value []byte, expiration time.Duration) error
}

// PubSubSubscriber abstracts Redis Pub/Sub for event-driven cache invalidation.
// The real implementation wraps *redis.Client; tests can provide a mock.
type PubSubSubscriber interface {
	// Subscribe returns a channel that emits messages published to the given channel.
	// The returned cancel function must be called to unsubscribe and clean up.
	Subscribe(ctx context.Context, channel string) (<-chan string, func(), error)
}

// CachedRole is a single registration role stored in Redis.
type CachedRole struct {
	Code        string `json:"code"`
	DisplayName string `json:"display_name"`
	JobTypeID   string `json:"job_type_id,omitempty"` // Non-empty only for tenant-specific roles
}

// Cache manages the role cache lifecycle.
type Cache struct {
	store        RedisStore
	pubsub       PubSubSubscriber
	routingTable *routing.Table
	apiClient    *backend.Client
	keyPrefix    string
	logger       *slog.Logger
}

// NewCache creates a new role cache.
func NewCache(
	store RedisStore,
	routeTable *routing.Table,
	apiClient *backend.Client,
	logger *slog.Logger,
) *Cache {
	if logger == nil {
		logger = slog.Default()
	}
	return &Cache{
		store:        store,
		routingTable: routeTable,
		apiClient:    apiClient,
		keyPrefix:    "ussd:roles:",
		logger:       logger,
	}
}

// SetPubSub attaches a Pub/Sub subscriber for event-driven invalidation.
// Called after NewCache since Pub/Sub is optional (graceful degradation).
func (c *Cache) SetPubSub(ps PubSubSubscriber) {
	c.pubsub = ps
}

// InvalidateChannel is the Redis Pub/Sub channel name for cache invalidation events.
const InvalidateChannel = "ussd:role_cache:invalidate"

// GetRoles returns registration roles for a service code.
//
// Resolution order:
//  1. Redis cache (sub-ms) — populated by background cron
//  2. Hardcoded fallback (0ms) — compiled into the binary, never wrong for the industry
//
// This method NEVER makes an API call and NEVER returns an error.
func (c *Cache) GetRoles(ctx context.Context, serviceCode string) []CachedRole {
	route := c.routingTable.Resolve(serviceCode)
	cacheKey := c.key(serviceCode)

	// Layer 1: Redis cache
	data, err := c.store.GetBytes(ctx, cacheKey)
	if err == nil && len(data) > 0 {
		var roles []CachedRole
		if json.Unmarshal(data, &roles) == nil && len(roles) > 0 {
			metrics.RoleCacheHitsTotal.WithLabelValues(serviceCode).Inc()
			c.logger.Debug("role cache hit",
				slog.String("service_code", serviceCode),
				slog.Int("roles", len(roles)),
			)
			return roles
		}
	}

	// Layer 2: Hardcoded fallback
	metrics.RoleCacheMissesTotal.WithLabelValues(serviceCode).Inc()
	c.logger.Debug("role cache miss, using hardcoded fallback",
		slog.String("service_code", serviceCode),
		slog.String("industry", route.IndustryType),
	)
	return c.hardcodedRoles(route.IndustryType)
}

// RefreshAll fetches roles from the backend API for every configured service code
// and writes them to Redis. Called by the background cron and on startup.
//
// For each service code:
//   - If the route has an @org_id → fetch tenant-specific job types (Level 2 override)
//   - If tenant fetch fails or no org_id → fetch industry template defaults (Level 1)
//   - If API is completely unreachable → skip (existing cache or hardcoded fallback remains)
func (c *Cache) RefreshAll(ctx context.Context) {
	start := time.Now()
	routes := c.routingTable.AllRoutes()
	if len(routes) == 0 {
		return
	}

	refreshed := 0
	failed := 0

	for _, route := range routes {
		c.refreshServiceCode(ctx, route, &refreshed, &failed)
	}

	// Record refresh cycle duration and timestamp
	metrics.RoleCacheRefreshDuration.Observe(time.Since(start).Seconds())
	if refreshed > 0 {
		metrics.RoleCacheLastRefreshTimestamp.SetToCurrentTime()
	}

	c.logger.Info("role cache refresh complete",
		slog.Int("refreshed", refreshed),
		slog.Int("failed", failed),
		slog.Int("total", len(routes)),
		slog.Duration("duration", time.Since(start)),
	)
}

// RefreshServiceCode refreshes the cache for a single service code.
// Used by Pub/Sub invalidation for targeted cache updates.
func (c *Cache) RefreshServiceCode(ctx context.Context, serviceCode string) {
	route := c.routingTable.Resolve(serviceCode)
	refreshed, failed := 0, 0
	c.refreshServiceCode(ctx, route, &refreshed, &failed)

	if refreshed > 0 {
		metrics.RoleCacheLastRefreshTimestamp.SetToCurrentTime()
	}
}

// refreshServiceCode is the shared logic for refreshing a single route's cache.
func (c *Cache) refreshServiceCode(ctx context.Context, route routing.Route, refreshed, failed *int) {
	var roles []CachedRole

	// Level 2: Tenant-specific override
	if route.OrganizationID != "" {
		roles = c.fetchTenantRoles(ctx, route.OrganizationID)
	}

	// Level 1: Industry template (if tenant fetch failed or no org)
	if len(roles) == 0 {
		roles = c.fetchIndustryRoles(ctx, route.IndustryType)
	}

	if len(roles) == 0 {
		*failed++
		metrics.RoleCacheRefreshTotal.WithLabelValues(route.ServiceCode, "error").Inc()
		c.logger.Warn("cache refresh: no roles fetched, keeping existing cache",
			slog.String("service_code", route.ServiceCode),
			slog.String("industry", route.IndustryType),
		)
		return
	}

	// Write to Redis (no TTL — refreshed by cron, never expires)
	if err := c.setRoles(ctx, route.ServiceCode, roles); err != nil {
		*failed++
		metrics.RoleCacheRefreshTotal.WithLabelValues(route.ServiceCode, "error").Inc()
		c.logger.Error("cache refresh: failed to write to Redis",
			slog.String("service_code", route.ServiceCode),
			slog.String("error", err.Error()),
		)
		return
	}
	*refreshed++
	metrics.RoleCacheRefreshTotal.WithLabelValues(route.ServiceCode, "success").Inc()
}

// --- Background cron (midnight-aligned) ---

// StartCron starts the background cache refresh goroutine.
// Runs an initial refresh immediately, then aligns subsequent refreshes to midnight.
// The interval determines the period between refreshes (default 24h = once per midnight).
// Returns a cancel function to stop the cron.
func (c *Cache) StartCron(interval time.Duration) func() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		// Initial population on startup
		c.logger.Info("role cache: initial population starting")
		c.RefreshAll(ctx)

		// Start Pub/Sub listener for event-driven invalidation (non-blocking)
		c.startPubSubListener(ctx)

		// Align first tick to next midnight
		nextMidnight := c.nextMidnight(time.Now())
		untilMidnight := time.Until(nextMidnight)

		c.logger.Info("role cache: cron aligned to midnight",
			slog.Time("next_refresh", nextMidnight),
			slog.Duration("wait", untilMidnight),
			slog.Duration("interval", interval),
		)

		// Wait until midnight
		select {
		case <-ctx.Done():
			c.logger.Info("role cache cron stopped (pre-midnight)")
			return
		case <-time.After(untilMidnight):
			c.logger.Info("role cache: midnight refresh starting")
			c.RefreshAll(ctx)
		}

		// Then tick at the configured interval
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				c.logger.Info("role cache cron stopped")
				return
			case <-ticker.C:
				c.logger.Info("role cache: scheduled refresh starting")
				c.RefreshAll(ctx)
			}
		}
	}()

	return cancel
}

// nextMidnight returns the next midnight (00:00:00) in the local timezone.
func (c *Cache) nextMidnight(now time.Time) time.Time {
	tomorrow := now.Add(24 * time.Hour)
	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, now.Location())
}

// --- Redis Pub/Sub listener ---

// startPubSubListener subscribes to the invalidation channel and refreshes
// specific service codes when events are published. If no PubSubSubscriber
// is configured, this is a no-op (graceful degradation).
//
// Payload format:
//   - "*"           → refresh all service codes
//   - "<code>"      → refresh a specific service code (e.g., "*384*200#")
func (c *Cache) startPubSubListener(ctx context.Context) {
	if c.pubsub == nil {
		c.logger.Debug("role cache: pub/sub not configured, skipping event-driven invalidation")
		return
	}

	messages, unsub, err := c.pubsub.Subscribe(ctx, InvalidateChannel)
	if err != nil {
		c.logger.Error("role cache: failed to subscribe to invalidation channel",
			slog.String("channel", InvalidateChannel),
			slog.String("error", err.Error()),
		)
		return
	}

	c.logger.Info("role cache: pub/sub listener started",
		slog.String("channel", InvalidateChannel),
	)

	go func() {
		defer unsub()
		for {
			select {
			case <-ctx.Done():
				c.logger.Info("role cache: pub/sub listener stopped")
				return
			case msg, ok := <-messages:
				if !ok {
					c.logger.Warn("role cache: pub/sub channel closed, listener exiting")
					return
				}
				c.handleInvalidation(ctx, msg)
			}
		}
	}()
}

// handleInvalidation processes a single invalidation message.
func (c *Cache) handleInvalidation(ctx context.Context, payload string) {
	if payload == "*" {
		c.logger.Info("role cache: pub/sub received full invalidation")
		c.RefreshAll(ctx)
		return
	}

	// Targeted invalidation for a single service code
	if c.routingTable.HasRoute(payload) {
		c.logger.Info("role cache: pub/sub invalidating service code",
			slog.String("service_code", payload),
		)
		c.RefreshServiceCode(ctx, payload)
	} else {
		c.logger.Warn("role cache: pub/sub received unknown service code, ignoring",
			slog.String("payload", payload),
		)
	}
}

// --- Internal helpers ---

func (c *Cache) key(serviceCode string) string {
	return c.keyPrefix + serviceCode
}

func (c *Cache) setRoles(ctx context.Context, serviceCode string, roles []CachedRole) error {
	data, err := json.Marshal(roles)
	if err != nil {
		return fmt.Errorf("marshal roles: %w", err)
	}
	// No TTL — cache is refreshed by cron, never auto-expires.
	// This means even if the cron fails, stale but correct data persists.
	return c.store.SetBytes(ctx, c.key(serviceCode), data, 0)
}

func (c *Cache) hardcodedRoles(industryType string) []CachedRole {
	jts := routing.GetIndustryJobTypes(industryType)
	roles := make([]CachedRole, len(jts))
	for i, jt := range jts {
		roles[i] = CachedRole{
			Code:        jt.Code,
			DisplayName: jt.DisplayName,
		}
	}
	return roles
}

func (c *Cache) fetchTenantRoles(ctx context.Context, orgID string) []CachedRole {
	if c.apiClient == nil {
		return nil
	}
	jobTypes, err := c.apiClient.GetOrganizationJobTypes(ctx, orgID)
	if err != nil {
		c.logger.Debug("tenant role fetch failed",
			slog.String("org_id", orgID),
			slog.String("error", err.Error()),
		)
		return nil
	}

	var roles []CachedRole
	for _, jt := range jobTypes {
		if !jt.IsActive {
			continue
		}
		roles = append(roles, CachedRole{
			Code:        jt.Code,
			DisplayName: jt.DisplayName,
			JobTypeID:   jt.ID,
		})
	}
	return roles
}

func (c *Cache) fetchIndustryRoles(ctx context.Context, industryType string) []CachedRole {
	if c.apiClient == nil {
		return nil
	}
	tmpl, err := c.apiClient.GetIndustryTemplate(ctx, industryType)
	if err != nil || tmpl == nil {
		c.logger.Debug("industry template fetch failed",
			slog.String("industry", industryType),
		)
		return nil
	}

	var roles []CachedRole
	for _, jt := range tmpl.DefaultJobTypes {
		// Only PRIMARY and FACILITATOR for self-registration
		if jt.Category == "SUPERVISOR" || jt.Category == "SUPPORT" {
			continue
		}
		roles = append(roles, CachedRole{
			Code:        jt.Code,
			DisplayName: jt.DisplayName,
		})
	}
	return roles
}

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// PermissionCache provides Redis-backed permission caching for fast auth checks.
type PermissionCache struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewPermissionCache creates a new permission cache.
// If rdb is nil, the cache operates in passthrough mode (always misses).
func NewPermissionCache(rdb *redis.Client, ttl time.Duration) *PermissionCache {
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	return &PermissionCache{rdb: rdb, ttl: ttl}
}

func (c *PermissionCache) cacheKey(userID uuid.UUID, tenantID *uuid.UUID) string {
	tid := "global"
	if tenantID != nil {
		tid = tenantID.String()
	}
	return fmt.Sprintf("rbac:perms:%s:%s", userID.String(), tid)
}

// Get retrieves cached permission keys for a user+tenant.
// Returns nil, false on cache miss or if Redis is unavailable.
func (c *PermissionCache) Get(ctx context.Context, userID uuid.UUID, tenantID *uuid.UUID) ([]string, bool) {
	if c.rdb == nil {
		return nil, false
	}

	data, err := c.rdb.Get(ctx, c.cacheKey(userID, tenantID)).Bytes()
	if err != nil {
		return nil, false
	}

	var keys []string
	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, false
	}
	return keys, true
}

// Set stores permission keys in the cache.
func (c *PermissionCache) Set(ctx context.Context, userID uuid.UUID, tenantID *uuid.UUID, keys []string) {
	if c.rdb == nil {
		return
	}

	data, err := json.Marshal(keys)
	if err != nil {
		slog.Warn("rbac cache: marshal error", slog.Any("error", err))
		return
	}

	if err := c.rdb.Set(ctx, c.cacheKey(userID, tenantID), data, c.ttl).Err(); err != nil {
		slog.Warn("rbac cache: set error", slog.Any("error", err))
	}
}

// Invalidate removes cached permissions for a user (all tenants via pattern).
func (c *PermissionCache) Invalidate(ctx context.Context, userID uuid.UUID) {
	if c.rdb == nil {
		return
	}
	pattern := fmt.Sprintf("rbac:perms:%s:*", userID.String())
	iter := c.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		c.rdb.Del(ctx, iter.Val())
	}
}

// InvalidateForTenant removes cached permissions for all users in a tenant.
func (c *PermissionCache) InvalidateForTenant(ctx context.Context, tenantID uuid.UUID) {
	if c.rdb == nil {
		return
	}
	pattern := fmt.Sprintf("rbac:perms:*:%s", tenantID.String())
	iter := c.rdb.Scan(ctx, 0, pattern, 500).Iterator()
	for iter.Next(ctx) {
		c.rdb.Del(ctx, iter.Val())
	}
}

// InvalidateAll clears the entire RBAC permission cache.
func (c *PermissionCache) InvalidateAll(ctx context.Context) {
	if c.rdb == nil {
		return
	}
	pattern := "rbac:perms:*"
	iter := c.rdb.Scan(ctx, 0, pattern, 1000).Iterator()
	for iter.Next(ctx) {
		c.rdb.Del(ctx, iter.Val())
	}
}

// HasPermission checks if a specific permission key exists in the cached set.
func (c *PermissionCache) HasPermission(ctx context.Context, userID uuid.UUID, tenantID *uuid.UUID, permKey string) (allowed bool, cached bool) {
	keys, ok := c.Get(ctx, userID, tenantID)
	if !ok {
		return false, false
	}
	for _, k := range keys {
		if k == permKey {
			return true, true
		}
	}
	return false, true
}

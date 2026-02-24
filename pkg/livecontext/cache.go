package livecontext

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// freshTTL is how long a cached entry is considered fresh (served without re-fetching).
	freshTTL = 30 * time.Second
	// staleTTL is how long a cached entry is kept in Redis as a stale fallback.
	staleTTL = 5 * time.Minute
)

// Cache wraps Redis for live context data with a 30s fresh window
// and 5-minute stale fallback for graceful degradation.
type Cache struct {
	rdb *redis.Client
}

func NewCache(rdb *redis.Client) *Cache {
	return &Cache{rdb: rdb}
}

type cachedEntry struct {
	Data     json.RawMessage `json:"data"`
	CachedAt time.Time       `json:"cached_at"`
}

// IsFresh returns true if the entry is within the 30s fresh window.
func (e *cachedEntry) IsFresh() bool {
	return time.Since(e.CachedAt) < freshTTL
}

func (c *Cache) Get(ctx context.Context, key string) (*cachedEntry, error) {
	val, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("reading cache: %w", err)
	}
	var entry cachedEntry
	if err := json.Unmarshal(val, &entry); err != nil {
		return nil, fmt.Errorf("unmarshaling cache entry: %w", err)
	}
	return &entry, nil
}

func (c *Cache) Set(ctx context.Context, key string, data json.RawMessage) error {
	entry := cachedEntry{
		Data:     data,
		CachedAt: time.Now().UTC(),
	}
	val, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling cache entry: %w", err)
	}
	return c.rdb.Set(ctx, key, val, staleTTL).Err()
}

func OnCallKey(tenantSlug, rosterID string) string {
	return fmt.Sprintf("live-context:%s:oncall:%s", tenantSlug, rosterID)
}

func ServiceAlertsKey(tenantSlug, serviceName string) string {
	return fmt.Sprintf("live-context:%s:alerts:%s", tenantSlug, serviceName)
}

func ActiveAlertsKey(tenantSlug, severity string) string {
	return fmt.Sprintf("live-context:%s:alerts:active:%s", tenantSlug, severity)
}

func IncidentKey(tenantSlug, incidentID string) string {
	return fmt.Sprintf("live-context:%s:incident:%s", tenantSlug, incidentID)
}

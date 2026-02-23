package livecontext

import (
	"encoding/json"
	"time"
)

// TenantNightOwlConfig holds the NightOwl connection settings parsed from tenant config JSONB.
type TenantNightOwlConfig struct {
	APIURL string `json:"nightowl_api_url"`
	APIKey string `json:"nightowl_api_key"`
}

// Envelope is the standard response envelope for live context data.
type Envelope struct {
	Data     json.RawMessage `json:"data"`
	CachedAt *time.Time      `json:"cached_at,omitempty"`
	Source   string          `json:"source"` // "cache", "live", or "unavailable"
}

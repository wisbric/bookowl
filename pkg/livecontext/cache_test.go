package livecontext

import (
	"testing"
	"time"
)

func TestOnCallKey(t *testing.T) {
	tests := []struct {
		tenant   string
		rosterID string
		want     string
	}{
		{"acme", "primary", "live-context:acme:oncall:primary"},
		{"bigcorp", "secondary", "live-context:bigcorp:oncall:secondary"},
	}
	for _, tt := range tests {
		got := OnCallKey(tt.tenant, tt.rosterID)
		if got != tt.want {
			t.Errorf("OnCallKey(%q, %q) = %q, want %q", tt.tenant, tt.rosterID, got, tt.want)
		}
	}
}

func TestServiceAlertsKey(t *testing.T) {
	tests := []struct {
		tenant  string
		service string
		want    string
	}{
		{"acme", "api-gateway", "live-context:acme:alerts:api-gateway"},
		{"bigcorp", "db", "live-context:bigcorp:alerts:db"},
	}
	for _, tt := range tests {
		got := ServiceAlertsKey(tt.tenant, tt.service)
		if got != tt.want {
			t.Errorf("ServiceAlertsKey(%q, %q) = %q, want %q", tt.tenant, tt.service, got, tt.want)
		}
	}
}

func TestActiveAlertsKey(t *testing.T) {
	tests := []struct {
		tenant   string
		severity string
		want     string
	}{
		{"acme", "critical", "live-context:acme:alerts:active:critical"},
		{"bigcorp", "major", "live-context:bigcorp:alerts:active:major"},
	}
	for _, tt := range tests {
		got := ActiveAlertsKey(tt.tenant, tt.severity)
		if got != tt.want {
			t.Errorf("ActiveAlertsKey(%q, %q) = %q, want %q", tt.tenant, tt.severity, got, tt.want)
		}
	}
}

func TestIncidentKey(t *testing.T) {
	tests := []struct {
		tenant     string
		incidentID string
		want       string
	}{
		{"acme", "inc-123", "live-context:acme:incident:inc-123"},
		{"bigcorp", "456", "live-context:bigcorp:incident:456"},
	}
	for _, tt := range tests {
		got := IncidentKey(tt.tenant, tt.incidentID)
		if got != tt.want {
			t.Errorf("IncidentKey(%q, %q) = %q, want %q", tt.tenant, tt.incidentID, got, tt.want)
		}
	}
}

func TestCachedEntry_IsFresh(t *testing.T) {
	tests := []struct {
		name     string
		cachedAt time.Time
		want     bool
	}{
		{"just cached", time.Now().Add(-1 * time.Second), true},
		{"15 seconds ago", time.Now().Add(-15 * time.Second), true},
		{"29 seconds ago", time.Now().Add(-29 * time.Second), true},
		{"31 seconds ago", time.Now().Add(-31 * time.Second), false},
		{"1 minute ago", time.Now().Add(-1 * time.Minute), false},
		{"5 minutes ago", time.Now().Add(-5 * time.Minute), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &cachedEntry{CachedAt: tt.cachedAt}
			got := entry.IsFresh()
			if got != tt.want {
				t.Errorf("IsFresh() = %v, want %v (cached %v ago)", got, tt.want, time.Since(tt.cachedAt))
			}
		})
	}
}

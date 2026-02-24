package session

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIssueAndValidate(t *testing.T) {
	mgr := NewManager("test-secret-key-32-bytes-long!!!", 1*time.Hour)

	w := httptest.NewRecorder()
	err := mgr.Issue(w, "acme", "user-123", "admin", "local")
	if err != nil {
		t.Fatalf("Issue() error: %v", err)
	}

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie to be set")
	}

	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == CookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("cookie %q not found", CookieName)
	}

	if !sessionCookie.HttpOnly {
		t.Error("expected HttpOnly=true")
	}
	if !sessionCookie.Secure {
		t.Error("expected Secure=true")
	}
	if sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Error("expected SameSite=Strict")
	}

	claims, err := mgr.Validate(sessionCookie.Value)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if claims.Subject != "user-123" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "user-123")
	}
	if claims.Tenant != "acme" {
		t.Errorf("Tenant = %q, want %q", claims.Tenant, "acme")
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %q, want %q", claims.Role, "admin")
	}
	if claims.AuthMethod != "local" {
		t.Errorf("AuthMethod = %q, want %q", claims.AuthMethod, "local")
	}
}

func TestValidateInvalidToken(t *testing.T) {
	mgr := NewManager("test-secret-key-32-bytes-long!!!", 1*time.Hour)

	_, err := mgr.Validate("invalid-jwt-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestValidateWrongKey(t *testing.T) {
	mgr1 := NewManager("key-one-32-bytes-long-xxxxxxxx", 1*time.Hour)
	mgr2 := NewManager("key-two-32-bytes-long-xxxxxxxx", 1*time.Hour)

	w := httptest.NewRecorder()
	_ = mgr1.Issue(w, "acme", "user-1", "admin", "local")
	cookie := w.Result().Cookies()[0]

	_, err := mgr2.Validate(cookie.Value)
	if err == nil {
		t.Error("expected error when validating with wrong key")
	}
}

func TestShouldRefresh(t *testing.T) {
	mgr := NewManager("test-secret-key-32-bytes-long!!!", 1*time.Hour)

	tests := []struct {
		name     string
		ttl      time.Duration
		expected bool
	}{
		{"fresh token (12h TTL)", 12 * time.Hour, false},
		{"expiring soon (30min TTL)", 30 * time.Minute, true},
		{"just under refresh window", RefreshWindow - time.Minute, true},
		{"at refresh window boundary (1h TTL)", 1 * time.Hour, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortMgr := NewManager("test-secret-key-32-bytes-long!!!", tt.ttl)
			w := httptest.NewRecorder()
			_ = shortMgr.Issue(w, "acme", "user-1", "admin", "local")
			cookie := w.Result().Cookies()[0]

			claims, _ := mgr.Validate(cookie.Value)
			got := mgr.ShouldRefresh(claims)
			if got != tt.expected {
				t.Errorf("ShouldRefresh() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestClear(t *testing.T) {
	w := httptest.NewRecorder()
	Clear(w)

	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == CookieName {
			found = true
			if c.MaxAge != -1 {
				t.Errorf("MaxAge = %d, want -1", c.MaxAge)
			}
			if c.Value != "" {
				t.Errorf("Value = %q, want empty", c.Value)
			}
		}
	}
	if !found {
		t.Errorf("cookie %q not found in cleared response", CookieName)
	}
}

package session

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	CookieName    = "bw_session"
	RefreshWindow = 2 * time.Hour
)

// Claims are the JWT claims stored in the session cookie.
type Claims struct {
	jwt.RegisteredClaims
	Tenant     string `json:"tenant"`
	Role       string `json:"role"`
	AuthMethod string `json:"auth_method"`
	UserID     string `json:"user_id,omitempty"`
}

// Manager handles session JWT creation and validation.
type Manager struct {
	secretKey []byte
	ttl       time.Duration
}

// NewManager creates a session manager. The secretKey is used to sign JWTs with HMAC-SHA256.
func NewManager(secretKey string, ttl time.Duration) *Manager {
	return &Manager{
		secretKey: []byte(secretKey),
		ttl:       ttl,
	}
}

// Issue creates a new session JWT and sets it as a cookie.
func (m *Manager) Issue(w http.ResponseWriter, tenantSlug, userID, role, authMethod string) error {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			ID:        uuid.New().String(),
		},
		Tenant:     tenantSlug,
		Role:       role,
		AuthMethod: authMethod,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secretKey)
	if err != nil {
		return fmt.Errorf("signing session JWT: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    signed,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(m.ttl.Seconds()),
	})
	return nil
}

// Validate parses and validates a session JWT string.
func (m *Manager) Validate(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secretKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing session JWT: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid session JWT")
	}
	return claims, nil
}

// ShouldRefresh returns true if the session has less than RefreshWindow remaining.
func (m *Manager) ShouldRefresh(claims *Claims) bool {
	if claims.ExpiresAt == nil {
		return false
	}
	return time.Until(claims.ExpiresAt.Time) < RefreshWindow
}

// Clear removes the session cookie.
func Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

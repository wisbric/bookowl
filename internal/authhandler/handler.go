package authhandler

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	dbglobal "github.com/wisbric/bookowl/internal/db/global"
	"github.com/wisbric/bookowl/internal/httpserver"
	"github.com/wisbric/bookowl/internal/session"
)

const (
	maxLoginAttempts   = 10
	rateLimitWindow    = 15 * time.Minute
	bcryptCost         = 12
)

// Handler serves /auth/ endpoints (public, no tenant middleware).
type Handler struct {
	globalQ *dbglobal.Queries
	sess    *session.Manager
	rdb     *redis.Client
}

func NewHandler(globalDB dbglobal.DBTX, sess *session.Manager, rdb *redis.Client) *Handler {
	return &Handler{
		globalQ: dbglobal.New(globalDB),
		sess:    sess,
		rdb:     rdb,
	}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/local", h.LocalLogin)
	r.Post("/logout", h.Logout)
	r.Post("/change-password", h.ChangePassword)
	return r
}

// LocalLoginRequest is the body for POST /auth/local.
type LocalLoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	TenantSlug string `json:"tenant_slug"`
}

// LocalLoginResponse is returned on success.
type LocalLoginResponse struct {
	Redirect string `json:"redirect"`
}

func (h *Handler) LocalLogin(w http.ResponseWriter, r *http.Request) {
	var req LocalLoginRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Username == "" || req.Password == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "username and password required")
		return
	}

	tenantSlug := req.TenantSlug
	if tenantSlug == "" {
		tenantSlug = "acme" // default for single-tenant dev
	}

	// Rate limiting.
	ip := r.RemoteAddr
	rateLimitKey := fmt.Sprintf("login-attempts:%s", ip)
	count, _ := h.rdb.Get(r.Context(), rateLimitKey).Int()
	if count >= maxLoginAttempts {
		ttl, _ := h.rdb.TTL(r.Context(), rateLimitKey).Result()
		httpserver.Respond(w, http.StatusTooManyRequests, map[string]any{
			"error":       "too_many_attempts",
			"retry_after": int(ttl.Seconds()),
		})
		return
	}

	admin, err := h.globalQ.GetLocalAdmin(r.Context(), dbglobal.GetLocalAdminParams{
		TenantSlug: tenantSlug,
		Username:   req.Username,
	})
	if err != nil {
		h.recordFailedAttempt(r, rateLimitKey)
		httpserver.RespondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		h.recordFailedAttempt(r, rateLimitKey)
		httpserver.RespondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Clear rate limit on success.
	h.rdb.Del(r.Context(), rateLimitKey)

	// Update last login.
	if err := h.globalQ.UpdateLocalAdminLastLogin(r.Context(), admin.ID); err != nil {
		slog.Error("updating last login", "error", err)
	}

	// Check must_change.
	if admin.MustChange {
		// Issue a temporary session for the change-password page.
		if err := h.sess.Issue(w, tenantSlug, admin.ID.String(), "admin", "local"); err != nil {
			slog.Error("issuing session for password change", "error", err)
			httpserver.RespondError(w, http.StatusInternalServerError, "session error")
			return
		}
		httpserver.Respond(w, http.StatusForbidden, map[string]any{
			"error":    "must_change_password",
			"redirect": "/change-password",
		})
		return
	}

	// Issue session JWT.
	if err := h.sess.Issue(w, tenantSlug, admin.ID.String(), "admin", "local"); err != nil {
		slog.Error("issuing session", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "session error")
		return
	}

	httpserver.Respond(w, http.StatusOK, LocalLoginResponse{Redirect: "/"})
}

func (h *Handler) recordFailedAttempt(r *http.Request, key string) {
	pipe := h.rdb.Pipeline()
	pipe.Incr(r.Context(), key)
	pipe.Expire(r.Context(), key, rateLimitWindow)
	_, _ = pipe.Exec(r.Context())
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	session.Clear(w)
	httpserver.Respond(w, http.StatusOK, map[string]string{"redirect": "/login"})
}

// ChangePasswordRequest is the body for POST /auth/change-password.
type ChangePasswordRequest struct {
	NewPassword string `json:"new_password"`
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	// Must have a valid session.
	cookie, err := r.Cookie(session.CookieName)
	if err != nil {
		httpserver.RespondError(w, http.StatusUnauthorized, "session required")
		return
	}

	claims, err := h.sess.Validate(cookie.Value)
	if err != nil {
		httpserver.RespondError(w, http.StatusUnauthorized, "invalid session")
		return
	}

	if claims.AuthMethod != "local" {
		httpserver.RespondError(w, http.StatusForbidden, "password change only available for local admin")
		return
	}

	var req ChangePasswordRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validatePassword(req.NewPassword); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcryptCost)
	if err != nil {
		httpserver.RespondError(w, http.StatusInternalServerError, "password hashing failed")
		return
	}

	adminID, err := uuid.Parse(claims.Subject)
	if err != nil {
		httpserver.RespondError(w, http.StatusInternalServerError, "invalid session subject")
		return
	}

	_, err = h.globalQ.UpdateLocalAdminPassword(r.Context(), dbglobal.UpdateLocalAdminPasswordParams{
		ID:           adminID,
		PasswordHash: string(hash),
		MustChange:   false,
	})
	if err != nil {
		slog.Error("updating password", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	// Re-issue session (clear must_change state).
	if err := h.sess.Issue(w, claims.Tenant, claims.Subject, claims.Role, claims.AuthMethod); err != nil {
		slog.Error("re-issuing session after password change", "error", err)
	}

	httpserver.Respond(w, http.StatusOK, map[string]string{"status": "updated"})
}

func validatePassword(pw string) error {
	if len(pw) < 12 {
		return fmt.Errorf("password must be at least 12 characters")
	}
	hasUpper, hasLower, hasDigitOrSymbol := false, false, false
	for _, c := range pw {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasDigitOrSymbol = true
		default:
			hasDigitOrSymbol = true
		}
	}
	if !hasUpper || !hasLower {
		return fmt.Errorf("password must contain uppercase and lowercase letters")
	}
	if !hasDigitOrSymbol {
		return fmt.Errorf("password must contain a number or symbol")
	}
	return nil
}

// HashPassword hashes a password with bcrypt. Exported for use in seed.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}
	return string(hash), nil
}


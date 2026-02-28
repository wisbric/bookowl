package admin

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/wisbric/core/pkg/httpserver"

	dbglobal "github.com/wisbric/bookowl/internal/db/global"
	"github.com/wisbric/bookowl/pkg/tenant"
)

// OIDCConfig is the "oidc" section stored in tenant config JSONB.
type OIDCConfig struct {
	IssuerURL      string   `json:"issuer_url"`
	ClientID       string   `json:"client_id"`
	ClientSecret   string   `json:"client_secret"`
	AllowedGroups  []string `json:"allowed_groups,omitempty"`
	AdminGroups    []string `json:"admin_groups,omitempty"`
	EditorGroups   []string `json:"editor_groups,omitempty"`
	Enabled        bool     `json:"enabled"`
	LastTestedAt   string   `json:"last_tested_at,omitempty"`
	LastTestResult string   `json:"last_test_result,omitempty"`
}

// OIDCConfigResponse is returned by GET /admin/config/oidc (secret masked).
type OIDCConfigResponse struct {
	IssuerURL      string   `json:"issuer_url"`
	ClientID       string   `json:"client_id"`
	ClientSecret   string   `json:"client_secret"`
	AllowedGroups  []string `json:"allowed_groups"`
	AdminGroups    []string `json:"admin_groups"`
	EditorGroups   []string `json:"editor_groups"`
	Enabled        bool     `json:"enabled"`
	LastTestedAt   string   `json:"last_tested_at,omitempty"`
	LastTestResult string   `json:"last_test_result,omitempty"`
	Source         string   `json:"source,omitempty"` // "environment" when from env vars
}

// OIDCConfigUpdateRequest is the body for PUT /admin/config/oidc.
type OIDCConfigUpdateRequest struct {
	IssuerURL     string   `json:"issuer_url"`
	ClientID      string   `json:"client_id"`
	ClientSecret  string   `json:"client_secret,omitempty"`
	AllowedGroups []string `json:"allowed_groups"`
	AdminGroups   []string `json:"admin_groups"`
	EditorGroups  []string `json:"editor_groups"`
	Enabled       bool     `json:"enabled"`
}

// OIDCTestRequest is the body for POST /admin/config/oidc/test.
type OIDCTestRequest struct {
	IssuerURL    string `json:"issuer_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`
}

// OIDCTestResponse is returned by POST /admin/config/oidc/test.
type OIDCTestResponse struct {
	OK        bool            `json:"ok"`
	LatencyMs int64           `json:"latency_ms"`
	Issuer    string          `json:"issuer,omitempty"`
	Error     string          `json:"error,omitempty"`
	Details   OIDCTestDetails `json:"details"`
}

// OIDCTestDetails contains per-check results for an OIDC connection test.
type OIDCTestDetails struct {
	DiscoveryOK         bool     `json:"discovery_ok"`
	JWKSURI             string   `json:"jwks_uri,omitempty"`
	JWKSOK              bool     `json:"jwks_ok"`
	ClientCredentialsOK bool     `json:"client_credentials_ok"`
	SupportedScopes     []string `json:"supported_scopes,omitempty"`
	HasGroupsScope      bool     `json:"has_groups_scope"`
}

// encryptSecret encrypts a plaintext secret using AES-256-GCM.
func encryptSecret(plaintext, hexKey string) (string, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return "", fmt.Errorf("decoding secret key: %w", err)
	}
	if len(key) != 32 {
		return "", fmt.Errorf("secret key must be 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return "ENCRYPTED:AES256:" + hex.EncodeToString(ciphertext), nil
}

// decryptSecret decrypts an AES-256-GCM encrypted secret.
func decryptSecret(encrypted, hexKey string) (string, error) {
	if !strings.HasPrefix(encrypted, "ENCRYPTED:AES256:") {
		return encrypted, nil // not encrypted, return as-is
	}

	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return "", fmt.Errorf("decoding secret key: %w", err)
	}
	if len(key) != 32 {
		return "", fmt.Errorf("secret key must be 32 bytes, got %d", len(key))
	}

	data, err := hex.DecodeString(strings.TrimPrefix(encrypted, "ENCRYPTED:AES256:"))
	if err != nil {
		return "", fmt.Errorf("decoding ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypting: %w", err)
	}

	return string(plaintext), nil
}

// maskSecret returns a masked version of a secret for display.
func maskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	return "bw_****...****"
}

// loadOIDCConfig extracts the OIDC config from a tenant's config JSONB.
func loadOIDCConfig(configJSON json.RawMessage) OIDCConfig {
	var full map[string]json.RawMessage
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &full); err != nil {
			return OIDCConfig{}
		}
	}
	raw, ok := full["oidc"]
	if !ok {
		return OIDCConfig{}
	}
	var cfg OIDCConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return OIDCConfig{}
	}
	return cfg
}

// saveOIDCConfig merges the OIDC config into a tenant's config JSONB.
func saveOIDCConfig(existing json.RawMessage, oidcCfg OIDCConfig) (json.RawMessage, error) {
	full := map[string]json.RawMessage{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &full); err != nil {
			return nil, fmt.Errorf("parsing existing config: %w", err)
		}
	}
	oidcJSON, err := json.Marshal(oidcCfg)
	if err != nil {
		return nil, fmt.Errorf("marshaling OIDC config: %w", err)
	}
	full["oidc"] = oidcJSON
	return json.Marshal(full)
}

// GetOIDCConfig handles GET /admin/config/oidc.
func (h *Handler) GetOIDCConfig(w http.ResponseWriter, r *http.Request) {
	t, ok := tenant.FromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, http.StatusInternalServerError, "error", "tenant not resolved")
		return
	}

	cfg := loadOIDCConfig(t.Config)

	// If no OIDC section in tenant config, fall back to env-var defaults.
	source := ""
	if cfg.IssuerURL == "" && h.oidcDefaults.IssuerURL != "" {
		cfg.IssuerURL = h.oidcDefaults.IssuerURL
		cfg.ClientID = h.oidcDefaults.ClientID
		cfg.ClientSecret = "set-via-env"
		cfg.Enabled = true
		source = "environment"
	}

	httpserver.Respond(w, http.StatusOK, OIDCConfigResponse{
		IssuerURL:      cfg.IssuerURL,
		ClientID:       cfg.ClientID,
		ClientSecret:   maskSecret(cfg.ClientSecret),
		AllowedGroups:  nonNilSlice(cfg.AllowedGroups),
		AdminGroups:    nonNilSlice(cfg.AdminGroups),
		EditorGroups:   nonNilSlice(cfg.EditorGroups),
		Enabled:        cfg.Enabled,
		LastTestedAt:   cfg.LastTestedAt,
		LastTestResult: cfg.LastTestResult,
		Source:         source,
	})
}

// UpdateOIDCConfig handles PUT /admin/config/oidc.
func (h *Handler) UpdateOIDCConfig(w http.ResponseWriter, r *http.Request) {
	var req OIDCConfigUpdateRequest
	if !httpserver.DecodeAndValidate(w, r, &req) {

		return
	}

	t, ok := tenant.FromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, http.StatusInternalServerError, "error", "tenant not resolved")
		return
	}

	existing := loadOIDCConfig(t.Config)

	// Encrypt client secret if provided, otherwise keep existing.
	clientSecret := existing.ClientSecret
	if req.ClientSecret != "" {
		if h.secretKey == "" {
			httpserver.RespondError(w, http.StatusInternalServerError, "error", "BOOKOWL_SECRET_KEY not configured")
			return
		}
		encrypted, err := encryptSecret(req.ClientSecret, h.secretKey)
		if err != nil {
			slog.Error("encrypting client secret", "error", err)
			httpserver.RespondError(w, http.StatusInternalServerError, "error", "failed to encrypt client secret")
			return
		}
		clientSecret = encrypted
	}

	newCfg := OIDCConfig{
		IssuerURL:      req.IssuerURL,
		ClientID:       req.ClientID,
		ClientSecret:   clientSecret,
		AllowedGroups:  req.AllowedGroups,
		AdminGroups:    req.AdminGroups,
		EditorGroups:   req.EditorGroups,
		Enabled:        req.Enabled,
		LastTestedAt:   existing.LastTestedAt,
		LastTestResult: existing.LastTestResult,
	}

	configJSON, err := saveOIDCConfig(t.Config, newCfg)
	if err != nil {
		httpserver.RespondError(w, http.StatusInternalServerError, "error", "failed to serialize config")
		return
	}

	_, err = h.globalQ.UpdateTenant(r.Context(), dbglobal.UpdateTenantParams{
		ID:     t.ID,
		Name:   t.Name,
		Config: configJSON,
	})
	if err != nil {
		slog.Error("updating OIDC config", "tenant", t.Slug, "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "error", "failed to update config")
		return
	}

	// Invalidate Redis cache for this tenant's OIDC config.
	if h.redis != nil {
		cacheKey := fmt.Sprintf("oidc-config:%s", t.Slug)
		if err := h.redis.Del(r.Context(), cacheKey).Err(); err != nil {
			slog.Warn("failed to invalidate OIDC cache", "tenant", t.Slug, "error", err)
		}
	}

	httpserver.Respond(w, http.StatusOK, OIDCConfigResponse{
		IssuerURL:      newCfg.IssuerURL,
		ClientID:       newCfg.ClientID,
		ClientSecret:   maskSecret(newCfg.ClientSecret),
		AllowedGroups:  nonNilSlice(newCfg.AllowedGroups),
		AdminGroups:    nonNilSlice(newCfg.AdminGroups),
		EditorGroups:   nonNilSlice(newCfg.EditorGroups),
		Enabled:        newCfg.Enabled,
		LastTestedAt:   newCfg.LastTestedAt,
		LastTestResult: newCfg.LastTestResult,
	})
}

// TestOIDCConfig handles POST /admin/config/oidc/test.
func (h *Handler) TestOIDCConfig(w http.ResponseWriter, r *http.Request) {
	var req OIDCTestRequest
	if !httpserver.DecodeAndValidate(w, r, &req) {

		return
	}

	// Fall back to env-var defaults if not in request.
	if req.IssuerURL == "" {
		req.IssuerURL = h.oidcDefaults.IssuerURL
	}
	if req.ClientID == "" {
		req.ClientID = h.oidcDefaults.ClientID
	}

	if req.IssuerURL == "" {
		httpserver.Respond(w, http.StatusOK, OIDCTestResponse{
			OK:    false,
			Error: "issuer URL is required",
		})
		return
	}

	t, ok := tenant.FromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, http.StatusInternalServerError, "error", "tenant not resolved")
		return
	}

	// If no client secret in request, try to use the saved (encrypted) one.
	clientSecret := req.ClientSecret
	if clientSecret == "" && h.secretKey != "" {
		existing := loadOIDCConfig(t.Config)
		if existing.ClientSecret != "" {
			decrypted, err := decryptSecret(existing.ClientSecret, h.secretKey)
			if err != nil {
				slog.Warn("decrypting stored client secret for test", "error", err)
			} else {
				clientSecret = decrypted
			}
		}
	}

	start := time.Now()
	resp := testOIDCConnection(r.Context(), req.IssuerURL, req.ClientID, clientSecret)
	resp.LatencyMs = time.Since(start).Milliseconds()

	// Update last_tested_at on tenant config if we can.
	existing := loadOIDCConfig(t.Config)
	result := "ok"
	if !resp.OK {
		result = "error"
	}
	existing.LastTestedAt = time.Now().UTC().Format(time.RFC3339)
	existing.LastTestResult = result
	if configJSON, err := saveOIDCConfig(t.Config, existing); err == nil {
		_, _ = h.globalQ.UpdateTenant(r.Context(), dbglobal.UpdateTenantParams{
			ID:     t.ID,
			Name:   t.Name,
			Config: configJSON,
		})
	}

	httpserver.Respond(w, http.StatusOK, resp)
}

// testOIDCConnection performs the actual OIDC connection test.
func testOIDCConnection(ctx context.Context, issuerURL, clientID, clientSecret string) OIDCTestResponse {
	resp := OIDCTestResponse{
		Issuer: issuerURL,
	}

	// Step 1: Fetch discovery document.
	discoveryURL := strings.TrimRight(issuerURL, "/") + "/.well-known/openid-configuration"
	httpClient := &http.Client{Timeout: 10 * time.Second}
	discoveryReq, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		resp.Error = fmt.Sprintf("invalid issuer URL: %v", err)
		return resp
	}

	discoveryResp, err := httpClient.Do(discoveryReq)
	if err != nil {
		resp.Error = fmt.Sprintf("discovery document unreachable: %v", err)
		return resp
	}
	defer func() { _ = discoveryResp.Body.Close() }()

	if discoveryResp.StatusCode != http.StatusOK {
		resp.Error = fmt.Sprintf("discovery document returned %d", discoveryResp.StatusCode)
		return resp
	}

	var discovery struct {
		Issuer              string   `json:"issuer"`
		JWKSURI             string   `json:"jwks_uri"`
		TokenEndpoint       string   `json:"token_endpoint"`
		ScopesSupported     []string `json:"scopes_supported"`
		GrantTypesSupported []string `json:"grant_types_supported"`
	}
	if err := json.NewDecoder(discoveryResp.Body).Decode(&discovery); err != nil {
		resp.Error = "failed to parse discovery document"
		resp.Details.DiscoveryOK = false
		return resp
	}

	resp.Details.DiscoveryOK = true
	resp.Details.SupportedScopes = discovery.ScopesSupported

	// Check for groups scope.
	for _, s := range discovery.ScopesSupported {
		if s == "groups" {
			resp.Details.HasGroupsScope = true
			break
		}
	}

	// Step 2: Check JWKS endpoint.
	if discovery.JWKSURI != "" {
		resp.Details.JWKSURI = discovery.JWKSURI
		jwksReq, err := http.NewRequestWithContext(ctx, http.MethodGet, discovery.JWKSURI, nil)
		if err == nil {
			jwksResp, err := httpClient.Do(jwksReq)
			if err == nil {
				_ = jwksResp.Body.Close()
				resp.Details.JWKSOK = jwksResp.StatusCode == http.StatusOK
			}
		}
	}

	// Step 3: Test client credentials (if secret provided).
	if clientSecret != "" && discovery.TokenEndpoint != "" {
		tokenReq, err := http.NewRequestWithContext(ctx, http.MethodPost, discovery.TokenEndpoint,
			strings.NewReader("grant_type=client_credentials"))
		if err == nil {
			tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			tokenReq.SetBasicAuth(clientID, clientSecret)
			tokenResp, err := httpClient.Do(tokenReq)
			if err == nil {
				_ = tokenResp.Body.Close()
				resp.Details.ClientCredentialsOK = tokenResp.StatusCode == http.StatusOK
			}
		}
	}

	// Determine overall status.
	if resp.Details.DiscoveryOK && resp.Details.JWKSOK {
		resp.OK = true
		if clientSecret != "" && !resp.Details.ClientCredentialsOK {
			resp.OK = false
			resp.Error = "client credentials rejected"
		}
	} else if !resp.Details.DiscoveryOK {
		resp.Error = "discovery document check failed"
	} else {
		resp.Error = "JWKS endpoint unreachable"
	}

	return resp
}

// nonNilSlice ensures a nil slice is returned as an empty array in JSON.
func nonNilSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

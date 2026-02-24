# BookOwl — Keycloak / OIDC Configuration

## 1. Overview

BookOwl's OIDC connection (Keycloak realm, client ID, client secret) is configurable via the admin UI without requiring a pod restart. Settings are stored in the tenant config JSONB column and take effect on the next request — no environment variable changes, no redeployment.

Environment variables (`BOOKOWL_OIDC_ISSUER`, `BOOKOWL_OIDC_CLIENT_ID`) serve as **bootstrap defaults** only. Once a tenant has saved OIDC config via the UI, the database values take precedence.

---

## 2. What Gets Configured

| Setting | Description | Example |
|---------|-------------|---------|
| Issuer URL | Keycloak realm URL | `https://keycloak.example.com/realms/wisbric` |
| Client ID | OAuth2 client identifier | `bookowl` |
| Client Secret | OAuth2 client secret (confidential client) | `abc123...` |
| Redirect URI | Where Keycloak sends the auth code (auto-derived, shown read-only) | `https://bookowl.example.com/auth/callback` |
| Allowed groups | Optional: restrict login to Keycloak groups | `bookowl-users, platform-team` |
| Admin groups | Keycloak groups that get BookOwl admin role | `bookowl-admins, platform-leads` |

### 2.1 Role Mapping

BookOwl maps Keycloak group membership to internal roles:

| Keycloak group (configurable) | BookOwl role |
|-------------------------------|--------------|
| `bookowl-admins` (default) | admin |
| `bookowl-editors` (default) | editor |
| everyone else (allowed) | viewer |

If no group config is set, all authenticated users get `viewer` role and admins must be promoted manually via the admin UI.

---

## 3. Data Model

Add to the tenant `config` JSONB column (no new table needed):

```json
{
  "oidc": {
    "issuer_url": "https://keycloak.example.com/realms/wisbric",
    "client_id": "bookowl",
    "client_secret": "ENCRYPTED:AES256:...",
    "allowed_groups": ["bookowl-users"],
    "admin_groups": ["bookowl-admins"],
    "editor_groups": ["bookowl-editors"],
    "enabled": true,
    "last_tested_at": "2026-02-24T11:00:00Z",
    "last_test_result": "ok"
  }
}
```

**Client secret encryption:** The client secret is encrypted at rest using AES-256-GCM with a key derived from `BOOKOWL_SECRET_KEY` (same key used for API key hashing). It is never returned in plaintext via the API — the GET endpoint returns a masked value (`bw_****...****`).

---

## 4. Backend

### 4.1 Config Endpoints

```
GET  /api/v1/admin/config/oidc
     → Returns current OIDC settings (client_secret masked)
     → Requires admin role

PUT  /api/v1/admin/config/oidc
     → Updates OIDC settings
     → Encrypts client_secret before storing
     → Does NOT reload the OIDC provider immediately — happens on next request
     → Returns updated config (secret masked)
     → Requires admin role

POST /api/v1/admin/config/oidc/test
     → Tests the connection with current (or submitted) settings
     → Fetches the OIDC discovery document from issuer_url + /.well-known/openid-configuration
     → Verifies client credentials via client_credentials grant (if secret provided)
     → Returns: { ok: bool, latency_ms: int, issuer: string, error?: string, details: OIDCTestDetails }
     → Does NOT save — test first, save separately
     → Requires admin role
```

### 4.2 Test Endpoint Response

```json
{
  "ok": true,
  "latency_ms": 42,
  "issuer": "https://keycloak.example.com/realms/wisbric",
  "details": {
    "discovery_ok": true,
    "jwks_uri": "https://keycloak.example.com/realms/wisbric/protocol/openid-connect/certs",
    "jwks_ok": true,
    "client_credentials_ok": true,
    "supported_scopes": ["openid", "profile", "email", "groups"],
    "has_groups_scope": true,
    "keycloak_version": "24.0.1"
  }
}
```

Error example:

```json
{
  "ok": false,
  "latency_ms": 5001,
  "error": "connection timeout",
  "details": {
    "discovery_ok": false,
    "jwks_ok": false,
    "client_credentials_ok": false
  }
}
```

### 4.3 OIDC Provider Hot Reload

The OIDC middleware holds a reference to the current provider config. When settings are updated via `PUT /api/v1/admin/config/oidc`, the middleware reloads on the next inbound request:

```go
// internal/auth/oidc.go

type OIDCMiddleware struct {
    mu       sync.RWMutex
    provider *oidc.Provider   // coreos/go-oidc v3
    verifier *oidc.IDTokenVerifier
    config   OIDCConfig
}

// Reload fetches a fresh OIDC provider from the issuer URL.
// Called after config update and on startup.
func (m *OIDCMiddleware) Reload(ctx context.Context, cfg OIDCConfig) error {
    provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
    if err != nil {
        return fmt.Errorf("creating OIDC provider: %w", err)
    }
    verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})
    m.mu.Lock()
    defer m.mu.Unlock()
    m.provider = provider
    m.verifier = verifier
    m.config = cfg
    return nil
}
```

The middleware reads tenant OIDC config from DB on first request per tenant (cached in Redis, TTL 5 minutes). After a UI config update, the cache is invalidated so the next request picks up the new settings.

### 4.4 Group Claim Extraction

Keycloak puts group membership in the JWT under a configurable claim. Default claim path: `groups` (requires the `groups` mapper in the Keycloak client config).

```go
func extractGroups(claims map[string]interface{}) []string {
    raw, ok := claims["groups"]
    if !ok {
        return nil
    }
    switch v := raw.(type) {
    case []interface{}:
        groups := make([]string, 0, len(v))
        for _, g := range v {
            if s, ok := g.(string); ok {
                groups = append(groups, s)
            }
        }
        return groups
    case []string:
        return v
    }
    return nil
}

func mapGroupsToRole(groups []string, cfg OIDCConfig) string {
    for _, g := range groups {
        for _, adminGroup := range cfg.AdminGroups {
            if g == adminGroup {
                return "admin"
            }
        }
    }
    for _, g := range groups {
        for _, editorGroup := range cfg.EditorGroups {
            if g == editorGroup {
                return "editor"
            }
        }
    }
    if len(cfg.AllowedGroups) > 0 {
        for _, g := range groups {
            for _, allowed := range cfg.AllowedGroups {
                if g == allowed {
                    return "viewer"
                }
            }
        }
        return ""  // not in any allowed group → deny
    }
    return "viewer"  // no group restrictions → everyone is viewer
}
```

---

## 5. Frontend

### 5.1 Admin UI Location

The OIDC config lives at `/admin` under a new "Authentication" tab alongside the existing NightOwl and API Keys sections.

Tab layout of the admin page:

```
[ NightOwl ]  [ Authentication ]  [ API Keys ]  [ Audit Log ]  [ System ]
```

### 5.2 Authentication Tab

```
┌─────────────────────────────────────────────────────────┐
│  Authentication — Keycloak / OIDC                       │
│                                                         │
│  Issuer URL                                             │
│  ┌──────────────────────────────────────────────────┐  │
│  │ https://keycloak.example.com/realms/wisbric      │  │
│  └──────────────────────────────────────────────────┘  │
│                                                         │
│  Client ID                  Client Secret              │
│  ┌──────────────────┐       ┌──────────────────────┐  │
│  │ bookowl          │       │ ••••••••••••••••     │  │
│  └──────────────────┘       └──────────────────────┘  │
│                               [ Reveal ]  [ Clear ]    │
│                                                         │
│  Redirect URI (read-only)                              │
│  ┌──────────────────────────────────────────────────┐  │
│  │ https://bookowl.example.com/auth/callback        │  │
│  └──────────────────────────────────────────────────┘  │
│  Copy this URL into your Keycloak client config.        │
│                                                         │
│  ──────────────────────────────────────────────────    │
│  Role Mapping                                           │
│                                                         │
│  Admin groups           Editor groups                  │
│  ┌──────────────────┐   ┌──────────────────────────┐  │
│  │ bookowl-admins   │   │ bookowl-editors          │  │
│  └──────────────────┘   └──────────────────────────┘  │
│  Comma-separated Keycloak group names                  │
│                                                         │
│  Allowed groups (leave empty to allow all)             │
│  ┌──────────────────────────────────────────────────┐  │
│  │ bookowl-users                                    │  │
│  └──────────────────────────────────────────────────┘  │
│                                                         │
│  ──────────────────────────────────────────────────    │
│                                                         │
│  [ Test Connection ]              [ Save Settings ]    │
│                                                         │
│  ✓ Connected · 42ms · Keycloak 24.0.1                  │
│    ✓ Discovery  ✓ JWKS  ✓ Client credentials           │
│    ✓ groups scope available                             │
└─────────────────────────────────────────────────────────┘
```

### 5.3 Connection Test Result States

**Pending (test in progress):**
```
⟳ Testing connection...
```

**Success:**
```
✓ Connected · 42ms · Keycloak 24.0.1
  ✓ Discovery document OK
  ✓ JWKS endpoint reachable
  ✓ Client credentials accepted
  ✓ groups scope available
```

**Partial failure (discovery OK but credentials wrong):**
```
⚠ Partially connected — check client secret
  ✓ Discovery document OK
  ✓ JWKS endpoint reachable
  ✗ Client credentials rejected (401)
  ✓ groups scope available
```

**Full failure:**
```
✗ Connection failed — issuer URL unreachable
  ✗ Discovery document: connection timeout (5001ms)
  – JWKS endpoint: not checked
  – Client credentials: not checked
```

**groups scope missing (warning, not error):**
```
✓ Connected · 38ms · Keycloak 24.0.1
  ✓ Discovery document OK
  ✓ JWKS endpoint reachable
  ✓ Client credentials accepted
  ⚠ groups scope not advertised — role mapping may not work
     Add a groups mapper to your Keycloak client
```

### 5.4 Keycloak Setup Guide (inline)

Below the test result, show a collapsible "Keycloak Setup Guide" with the minimal steps to configure the client:

```
▶ Keycloak Setup Guide

  1. In your Keycloak realm, go to Clients → Create client
  2. Client ID: bookowl
  3. Client authentication: ON (confidential client)
  4. Valid redirect URIs: https://bookowl.example.com/auth/callback
  5. Add client scope: Add a mapper of type "Group Membership"
     Token Claim Name: groups
     Full group path: OFF
  6. Copy the client secret from the Credentials tab
     and paste it into the Client Secret field above.
```

This saves the inevitable round-trip of "why doesn't role mapping work" — the groups mapper is almost always what people forget.

---

## 6. Keycloak Client Configuration Reference

For the operator setting up Keycloak (can be exported from the admin UI or via Terraform):

```json
{
  "clientId": "bookowl",
  "name": "BookOwl",
  "description": "BookOwl knowledge management platform",
  "enabled": true,
  "clientAuthenticatorType": "client-secret",
  "redirectUris": ["https://bookowl.example.com/auth/callback"],
  "webOrigins": ["https://bookowl.example.com"],
  "protocol": "openid-connect",
  "attributes": {
    "pkce.code.challenge.method": "S256"
  },
  "protocolMappers": [
    {
      "name": "groups",
      "protocol": "openid-connect",
      "protocolMapper": "oidc-group-membership-mapper",
      "config": {
        "claim.name": "groups",
        "full.path": "false",
        "access.token.claim": "true",
        "id.token.claim": "true",
        "userinfo.token.claim": "true"
      }
    }
  ]
}
```

---

## 7. Environment Variables (bootstrap only)

These are used on first startup before any UI config is saved. Once saved via UI, database values take precedence.

```bash
BOOKOWL_OIDC_ISSUER=https://keycloak.example.com/realms/wisbric
BOOKOWL_OIDC_CLIENT_ID=bookowl
BOOKOWL_OIDC_CLIENT_SECRET=changeme   # bootstrap only, rotate via UI
BOOKOWL_SECRET_KEY=<32-byte-hex>      # used to encrypt client secret at rest
```

---

## 8. Tasks

Add to `docs/08-tasks.md` as Phase 10:

### Phase 10: Keycloak / OIDC Admin UI

**Backend:**
- [ ] Add `oidc` section to tenant config JSONB schema
- [ ] AES-256-GCM encryption for client secret at rest (key from `BOOKOWL_SECRET_KEY`)
- [ ] `GET /api/v1/admin/config/oidc` — returns config with masked secret
- [ ] `PUT /api/v1/admin/config/oidc` — saves + encrypts secret, invalidates Redis cache
- [ ] `POST /api/v1/admin/config/oidc/test` — tests without saving, returns `OIDCTestDetails`
- [ ] OIDC middleware hot reload on config update (mutex-protected provider swap)
- [ ] Group claim extraction from JWT (`groups` claim)
- [ ] Group → role mapping (admin / editor / viewer / deny)
- [ ] Redis cache for tenant OIDC config (TTL 5 min, invalidated on save)
- [ ] Bootstrap: load OIDC config from env vars on first startup if DB config empty

**Frontend:**
- [ ] Add "Authentication" tab to admin page
- [ ] Issuer URL, Client ID, Client Secret (masked + reveal toggle) fields
- [ ] Redirect URI read-only field with copy button
- [ ] Role mapping fields: admin groups, editor groups, allowed groups
- [ ] "Test Connection" button → calls POST test endpoint, shows detailed result
- [ ] Connection test result with per-check status icons (discovery, JWKS, credentials, groups scope)
- [ ] "groups scope missing" warning with explanation
- [ ] Collapsible "Keycloak Setup Guide" section
- [ ] "Save Settings" button (separate from test)
- [ ] Show last test result + timestamp on page load

**Testing:**
- [ ] Unit tests for group → role mapping logic
- [ ] Unit tests for client secret encrypt/decrypt
- [ ] Unit tests for `OIDCTestDetails` parsing
- [ ] Integration test: PUT config → GET config → secret is masked
- [ ] Manual: update issuer URL via UI, verify next login uses new config without pod restart

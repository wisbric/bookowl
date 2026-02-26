# BookOwl — Login, Local Admin & User Profile

## 1. Overview

BookOwl has three auth entry points, all handled by the shared `core/pkg/auth` package:

| Method | Who uses it | How |
|--------|-------------|-----|
| OIDC (Keycloak) | All regular users | Redirect to Keycloak, back to `/auth/callback` → sets `wisbric_session` cookie |
| Local admin | Break-glass / initial setup | Username + password on the login page → sets `wisbric_session` cookie |
| API Key | Service-to-service (NightOwl) | `X-API-Key` header, no login page involved |

All browser sessions use the `wisbric_session` HttpOnly cookie (shared cookie name across NightOwl, BookOwl, and TicketOwl). The middleware automatically refreshes the cookie when the token has less than 2 hours remaining.

The local admin account exists so BookOwl is accessible when Keycloak is misconfigured or unavailable. It is a single account per tenant, not a full local user directory.

---

## 2. Login Page

Route: `/login` — public, no auth required.

```
┌─────────────────────────────────────────────────────┐
│                                                     │
│           🦉  BookOwl                               │
│           Where your operational knowledge lives.   │
│                                                     │
│  ┌───────────────────────────────────────────────┐  │
│  │                                               │  │
│  │   [ Sign in with Keycloak              →  ]   │  │
│  │                                               │  │
│  │   ─────────────── or ──────────────────       │  │
│  │                                               │  │
│  │   Username                                    │  │
│  │   ┌─────────────────────────────────────┐    │  │
│  │   │ admin                               │    │  │
│  │   └─────────────────────────────────────┘    │  │
│  │                                               │  │
│  │   Password                                    │  │
│  │   ┌─────────────────────────────────────┐    │  │
│  │   │ ••••••••                            │    │  │
│  │   └─────────────────────────────────────┘    │  │
│  │                                               │  │
│  │   [ Sign in ]                                 │  │
│  │                                               │  │
│  └───────────────────────────────────────────────┘  │
│                                                     │
│   BookOwl v0.1.0 · A Wisbric product                │
└─────────────────────────────────────────────────────┘
```

### 2.1 Behaviour

- **"Sign in with Keycloak" button** — redirects to Keycloak OIDC login. Button label is configurable: if OIDC issuer is not configured it shows greyed-out with tooltip "OIDC not configured — configure in Admin → Authentication".
- **Local admin form** — always visible below the OIDC button (separated by a divider). Even when OIDC is the primary method, the local admin form remains accessible for break-glass.
- On successful local login: redirect to the URL in `?return=` query param, defaulting to `/`.
- On failed local login: inline error "Invalid username or password" (no distinction between wrong user / wrong password).
- Rate limiting: 10 failed attempts per IP per 15 minutes → 429 with countdown timer shown in UI.
- The login page respects the dark mode default from the design system.
- No "Remember me" checkbox — sessions last `BOOKOWL_SESSION_TTL` (default 12h).

### 2.2 OIDC Callback

Route: `/auth/callback` — handles the OIDC authorization code flow:

1. Validate `state` parameter (CSRF protection, stored in session cookie)
2. Exchange code for tokens with Keycloak
3. Validate ID token (issuer, audience, expiry, signature)
4. Extract claims: `sub`, `email`, `preferred_username`, `given_name`, `family_name`, `groups`
5. Upsert user in `users` table (create on first login, update name/email on subsequent)
6. Map groups → BookOwl role (per `docs/11-oidc-admin.md`)
7. Issue session JWT (signed by `core/pkg/auth.SessionManager` using the shared secret key)
8. Set `wisbric_session` cookie (HttpOnly, Secure, SameSite=Strict)
9. Redirect to `?return=` or `/`

### 2.3 Session Token

The `core/pkg/auth.SessionManager` issues short-lived session JWTs rather than forwarding the Keycloak token to the frontend. This decouples the frontend from Keycloak and means token refresh is handled server-side.

```json
{
  "sub": "user-uuid",
  "tenant": "acme",
  "role": "admin",
  "auth_method": "oidc",
  "iat": 1740384000,
  "exp": 1740427200
}
```

Cookie: `wisbric_session` — HttpOnly, Secure, SameSite=Strict, Path=/

Refresh: The API automatically issues a new cookie when the token has less than 2h remaining (silent refresh, no frontend action needed).

---

## 3. Local Admin Account

### 3.1 Data Model

Stored in the global schema (not per-tenant), since it is a deployment-level break-glass account:

```sql
-- migrations/000002_create_local_admin.up.sql (global schema)

CREATE TABLE local_admins (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_slug     TEXT NOT NULL UNIQUE,
    username        TEXT NOT NULL DEFAULT 'admin',
    password_hash   TEXT NOT NULL,          -- bcrypt, cost 12
    must_change     BOOLEAN NOT NULL DEFAULT true,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 3.2 Default Password

On tenant creation (`make seed` or `--mode=seed`), a local admin account is created:

- Username: `admin`
- Default password: `bookowl-admin` (dev) or a randomly generated 16-char password (production seed)
- `must_change = true` always on creation

The seed command prints the generated password to stdout once and never again:

```
✓ Tenant "acme" created
✓ Local admin created
  Username: admin
  Password: Xk7mP9nQ2vR4sY8w   ← save this, it will not be shown again
```

Production seed (`--mode=seed`) uses `BOOKOWL_ADMIN_PASSWORD` env var if set, otherwise generates random.

### 3.3 Must-Change Password Flow

When `must_change = true` and the local admin logs in, they are redirected to `/change-password` before accessing the app:

```
┌──────────────────────────────────────────────────┐
│                                                  │
│  🦉  Change your password                        │
│                                                  │
│  You're using the default admin password.        │
│  Please set a new password before continuing.    │
│                                                  │
│  New password                                    │
│  ┌──────────────────────────────────────────┐   │
│  │                                          │   │
│  └──────────────────────────────────────────┘   │
│                                                  │
│  Confirm new password                            │
│  ┌──────────────────────────────────────────┐   │
│  │                                          │   │
│  └──────────────────────────────────────────┘   │
│                                                  │
│  Password requirements:                          │
│  ✓ At least 12 characters                        │
│  ✓ Contains uppercase and lowercase              │
│  ✓ Contains a number or symbol                   │
│                                                  │
│  [ Set password ]                                │
│                                                  │
└──────────────────────────────────────────────────┘
```

### 3.4 Local Login Endpoint

```
POST /auth/local
     Content-Type: application/json
     Body: { "username": "admin", "password": "..." }
     → 200: sets wisbric_session cookie, returns { "redirect": "/" }
     → 401: { "error": "invalid credentials" }
     → 403: { "error": "must_change_password", "redirect": "/change-password" }
     → 429: { "error": "too_many_attempts", "retry_after": 847 }
```

### 3.5 Logout

```
POST /auth/logout
     → Clears wisbric_session cookie
     → Redirects to /login
```

---

## 4. User Profile & Top-Right Menu

The top-right corner of the app shell shows the current user's avatar, name, and a dropdown menu.

### 4.1 Layout

```
┌─────────────────────────────────────────────────────────────────┐
│ 🦉 BookOwl    [Cmd+K Search...]                    [SK▾]        │
├─────────────────────────────────────────────────────────────────┤
│ Sidebar │                  Main content                          │
```

The `[SK▾]` is a circular avatar button showing the user's initials (or photo if set). Clicking it opens the user dropdown.

### 4.2 User Dropdown

```
                                        ┌────────────────────────┐
                                        │  Stefan Kloppenborg    │
                                        │  stefan@adfinis.de     │
                                        │  admin · via OIDC      │
                                        ├────────────────────────┤
                                        │  👤 Profile            │
                                        │  🔑 Personal tokens    │
                                        │  ⚙️  Admin             │ ← admins only
                                        │  🌙 Dark mode      ✓   │
                                        ├────────────────────────┤
                                        │  🔗 NightOwl           │
                                        ├────────────────────────┤
                                        │  ⎋  Sign out           │
                                        └────────────────────────┘
```

- **Name + email** — from OIDC claims or local admin username
- **Role badge** — `admin`, `editor`, or `viewer`
- **Auth method** — `via OIDC` or `local admin`
- **Profile** — view/edit display name, avatar, timezone
- **Personal tokens** — manage personal API tokens (see section 5)
- **Admin** — only shown to users with `role=admin`
- **Dark mode toggle** — persisted to localStorage
- **NightOwl link** — opens NightOwl in a new tab (URL from admin config)
- **Sign out** — calls `POST /auth/logout`

### 4.3 Avatar

Initials avatar (default): first letter of first name + first letter of last name, coloured by a hash of the user's ID using the design system palette. Example: "Stefan Kloppenborg" → `SK` on a teal background.

Photo avatar (optional): users can upload a profile photo from the Profile page. Stored as a storage object (same backend as document images). Max 2MB, circular crop in UI.

---

## 5. Personal Access Tokens

Personal tokens allow users to authenticate API calls from scripts and tools without exposing their OIDC credentials. They are scoped to the user's role and cannot exceed the user's own permissions.

### 5.1 UI — Personal Tokens Page

Route: `/profile/tokens`

```
┌─────────────────────────────────────────────────────────────┐
│  Personal Access Tokens                                     │
│                                                             │
│  Use personal tokens to authenticate API calls from        │
│  scripts, curl, or other tools.                             │
│                                                             │
│  [ + New token ]                                           │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │  runbook-sync-script                                 │  │
│  │  Created Feb 24 · Last used 2h ago                   │  │
│  │  Expires never                    [ Revoke ]         │  │
│  └─────────────────────────────────────────────────────┘  │
│  ┌─────────────────────────────────────────────────────┐  │
│  │  local-testing                                       │  │
│  │  Created Jan 12 · Never used                         │  │
│  │  Expires Mar 1, 2026              [ Revoke ]         │  │
│  └─────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 5.2 Create Token Dialog

```
┌────────────────────────────────────────────────────┐
│  New Personal Access Token                         │
│                                                    │
│  Token name                                        │
│  ┌──────────────────────────────────────────────┐ │
│  │ e.g. runbook-sync-script                     │ │
│  └──────────────────────────────────────────────┘ │
│                                                    │
│  Expiration                                        │
│  ○ Never                                           │
│  ● 30 days                                         │
│  ○ 90 days                                         │
│  ○ 1 year                                          │
│  ○ Custom date                                     │
│                                                    │
│  [ Cancel ]          [ Create token ]              │
└────────────────────────────────────────────────────┘
```

After creation, the token is shown **once**:

```
┌────────────────────────────────────────────────────┐
│  ✓ Token created                                   │
│                                                    │
│  bwp_sk_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9  │
│  [ Copy ]                                          │
│                                                    │
│  This token will not be shown again.               │
│  Store it somewhere safe.                          │
│                                                    │
│  [ Done ]                                          │
└────────────────────────────────────────────────────┘
```

### 5.3 Token Format

Personal tokens use a distinct prefix to distinguish them from service API keys:

- Service API keys: `bw_` prefix (created by admins for service accounts)
- Personal access tokens: `bwp_` prefix (created by users for personal use)

Both are validated by the same API key middleware — the prefix is just for human readability.

### 5.4 Data Model

```sql
-- In the tenant schema (not global — tokens are per-tenant)

ALTER TABLE api_keys ADD COLUMN user_id UUID REFERENCES users(id);
ALTER TABLE api_keys ADD COLUMN is_personal BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE api_keys ADD COLUMN expires_at TIMESTAMPTZ;
ALTER TABLE api_keys ADD COLUMN last_used_at TIMESTAMPTZ;
```

Personal tokens are stored in the existing `api_keys` table with `is_personal=true` and `user_id` set. They carry the user's current role at creation time — if the user's role changes, existing personal tokens retain the old role until revoked and recreated.

### 5.5 API Endpoints

```
GET    /api/v1/profile/tokens          — list current user's personal tokens
POST   /api/v1/profile/tokens          — create token, returns plaintext once
DELETE /api/v1/profile/tokens/:id      — revoke token
```

Admin endpoints (existing, unchanged):
```
GET    /api/v1/api-keys                — list all service API keys
POST   /api/v1/api-keys                — create service API key
DELETE /api/v1/api-keys/:id            — revoke service API key
```

---

## 6. Profile Page

Route: `/profile`

```
┌─────────────────────────────────────────────────────┐
│  Your Profile                                       │
│                                                     │
│       ┌────┐                                        │
│       │ SK │  Stefan Kloppenborg                   │
│       └────┘  stefan@adfinis.de                    │
│       [ Change photo ]                              │
│                                                     │
│  Display name                                       │
│  ┌───────────────────────────────────────────────┐  │
│  │ Stefan Kloppenborg                            │  │
│  └───────────────────────────────────────────────┘  │
│                                                     │
│  Timezone                                           │
│  ┌───────────────────────────────────────────────┐  │
│  │ Pacific/Auckland (UTC+13)                ▾    │  │
│  └───────────────────────────────────────────────┘  │
│                                                     │
│  Role                  Auth method                  │
│  admin                 via OIDC                     │
│                                                     │
│  [ Save changes ]                                   │
│                                                     │
│  ─────────────────────────────────────────────────  │
│  Change password        (local admin only)          │
│  ┌───────────────────────────────────────────────┐  │
│  │ Current password                              │  │
│  └───────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────┐  │
│  │ New password                                  │  │
│  └───────────────────────────────────────────────┘  │
│  [ Change password ]                                │
│                                                     │
└─────────────────────────────────────────────────────┘
```

Notes:
- Email is read-only (comes from OIDC claims or is fixed as "admin" for local admin)
- "Change password" section only shown for local admin users — OIDC users manage their password in Keycloak
- Timezone is stored in the user record and used for displaying timestamps throughout the app

---

## 7. Backend — New Endpoints

```
# Auth (public)
POST /auth/local                     — local admin login
POST /auth/logout                    — clear session
GET  /auth/oidc/login                — initiate OIDC redirect
GET  /auth/callback                  — OIDC callback handler
POST /auth/change-password           — change local admin password (requires valid session)

# Profile (authenticated)
GET  /api/v1/profile                 — current user info
PUT  /api/v1/profile                 — update display name, timezone
POST /api/v1/profile/avatar          — upload avatar image
GET  /api/v1/profile/tokens          — list personal tokens
POST /api/v1/profile/tokens          — create personal token
DELETE /api/v1/profile/tokens/:id    — revoke personal token
```

---

## 8. Database Changes

```sql
-- Add to users table
ALTER TABLE users ADD COLUMN timezone TEXT NOT NULL DEFAULT 'UTC';
ALTER TABLE users ADD COLUMN avatar_storage_id UUID REFERENCES storage_objects(id);
ALTER TABLE users ADD COLUMN display_name TEXT;
ALTER TABLE users ADD COLUMN auth_method TEXT NOT NULL DEFAULT 'oidc';  -- 'oidc' | 'local'

-- Add to api_keys table  
ALTER TABLE api_keys ADD COLUMN user_id UUID REFERENCES users(id);
ALTER TABLE api_keys ADD COLUMN is_personal BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE api_keys ADD COLUMN expires_at TIMESTAMPTZ;
ALTER TABLE api_keys ADD COLUMN last_used_at TIMESTAMPTZ;
```

---

## 9. Auth Middleware

Auth is handled by `core/pkg/auth.Middleware` (shared across all owl services). The middleware checks authentication methods in this order:

1. **Cookie** — `wisbric_session` HttpOnly cookie (OIDC or local admin session), with silent refresh when <2h remaining
2. **PAT** — Personal Access Token (`Authorization: Bearer bwp_...`)
3. **Session JWT** — `Authorization: Bearer <session-jwt>`
4. **OIDC JWT** — `Authorization: Bearer <oidc-token>`
5. **API Key** — `X-API-Key` header (service keys and personal tokens)
6. **Dev header** — `X-Tenant-Slug` header (`DEV_MODE=true` only)

BookOwl's auth storage adapter (`internal/authadapter/adapter.go`) implements `core/pkg/auth.Storage` and overrides `FindLocalAdmin` to handle BookOwl's `tenant_slug`-based `local_admins` table. OIDC group-to-role mapping is in `internal/authadapter/groups.go`.

---

## 10. Tasks

Add to `docs/08-tasks.md` as Phase 12:

### Phase 12: Login, Local Admin & User Profile

**Backend:**
- [ ] `local_admins` table migration (global schema)
- [ ] `ALTER TABLE users` — add timezone, avatar_storage_id, display_name, auth_method
- [ ] `ALTER TABLE api_keys` — add user_id, is_personal, expires_at, last_used_at
- [ ] `POST /auth/local` — bcrypt verify, rate limit (Redis), set session cookie
- [ ] `POST /auth/logout` — clear cookie
- [ ] `GET /auth/oidc/login` — OIDC redirect with state CSRF
- [ ] `GET /auth/callback` — token exchange, user upsert, group→role mapping, session cookie
- [ ] `POST /auth/change-password` — bcrypt new password, clear must_change flag
- [ ] Session JWT issue + validate (sign with BOOKOWL_SECRET_KEY)
- [ ] Silent session refresh (issue new cookie when <2h remaining)
- [ ] Rate limiter: 10 fails/IP/15min via Redis INCR + EXPIRE
- [ ] `GET/PUT /api/v1/profile` — user info + update display name/timezone
- [ ] `POST /api/v1/profile/avatar` — upload avatar via storage backend
- [ ] `GET/POST/DELETE /api/v1/profile/tokens` — personal token management
- [ ] Update auth middleware to handle session cookie + API key + dev fallback
- [ ] Seed: create local admin on tenant creation, print password once, set must_change=true
- [ ] Unit tests: bcrypt, rate limiter, session JWT, group→role, personal token CRUD

**Frontend:**
- [ ] `/login` page — OIDC button + local admin form, dark mode, rate limit countdown
- [ ] `/change-password` page — must-change flow after local admin first login
- [ ] OIDC callback route — handle `/auth/callback`, show loading spinner, redirect on success
- [ ] Auth context provider — wraps app, provides `useAuth()` hook with current user
- [ ] Redirect unauthenticated requests to `/login?return=<current-path>`
- [ ] Top-right user avatar button with initials + dropdown menu
- [ ] Dropdown: name, email, role badge, auth method, links, dark mode toggle, sign out
- [ ] `/profile` page — display name, timezone, avatar upload, change password (local only)
- [ ] `/profile/tokens` page — list, create (with one-time reveal), revoke personal tokens
- [ ] Token creation dialog with expiration options
- [ ] One-time token reveal with copy button and "will not be shown again" warning
